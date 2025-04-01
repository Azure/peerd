// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/azure/peerd/pkg/containerd"
	pcontext "github.com/azure/peerd/pkg/context"
	"github.com/azure/peerd/pkg/discovery/content/provider"
	"github.com/azure/peerd/pkg/discovery/routing"
	"github.com/azure/peerd/pkg/files/store"
	"github.com/azure/peerd/pkg/handlers"
	"github.com/azure/peerd/pkg/k8s"
	"github.com/azure/peerd/pkg/k8s/events"
	"github.com/azure/peerd/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/spf13/afero"
	"golang.org/x/sync/errgroup"
)

func main() {
	args := &Arguments{}
	arg.MustParse(args)

	ll, err := zerolog.ParseLevel(args.LogLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid log level: %s\n", args.LogLevel)
		os.Exit(1)
	}

	zerolog.SetGlobalLevel(ll)
	zerolog.TimeFieldFormat = time.RFC3339Nano

	l := zerolog.New(os.Stdout).With().Timestamp().Str("self", pcontext.NodeName).Str("version", version).Logger()
	ctx := l.WithContext(context.Background())

	ctx, err = metrics.WithContext(ctx, pcontext.NodeName, "peerd")
	if err != nil {
		l.Error().Err(err).Msg("failed to initialize metrics")
		os.Exit(1)
	}

	err = run(ctx, args)
	if err != nil {
		l.Error().Err(err).Msg("server error")
		os.Exit(1)
	}

	l.Info().Msg("server shutdown")
}

func run(ctx context.Context, args *Arguments) error {
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGTERM)
	defer cancel()

	switch {
	case args.Version:
		zerolog.Ctx(ctx).Info().Msg("version") // version field is already added to the logger
		return nil
	case args.Server != nil:
		return serverCommand(ctx, args.Server)
	default:
		return fmt.Errorf("unknown subcommand")
	}
}

func serverCommand(ctx context.Context, args *ServerCmd) (err error) {
	l := zerolog.Ctx(ctx)

	store.PrefetchWorkers = args.PrefetchWorkers

	_, httpsPort, err := net.SplitHostPort(args.HttpsAddr)
	if err != nil {
		return err
	}

	clientset, err := k8s.NewKubernetesInterface(pcontext.KubeConfigPath, pcontext.NodeName)
	if err != nil {
		return err
	}

	ctx, err = events.WithContext(ctx, clientset)
	if err != nil {
		return err
	}
	eventsRecorder := events.FromContext(ctx)
	defer func() {
		if err != nil {
			eventsRecorder.Failed()
		}
	}()
	eventsRecorder.Initializing()

	r, err := routing.NewRouter(ctx, clientset, args.RouterAddr, httpsPort)
	if err != nil {
		return err
	}

	err = addMirrorConfiguration(ctx, args)
	if err != nil {
		return err
	}

	containerdStore, err := containerd.NewDefaultStore(args.Hosts)
	if err != nil {
		return err
	}
	err = containerdStore.Verify(ctx)
	if err != nil {
		return err
	}

	filesStore, err := store.NewFilesStore(ctx, r, store.DefaultFileCachePath)
	if err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		provider.Provide(ctx, r, containerdStore, filesStore.Subscribe())
		return nil
	})

	handler, err := handlers.Handler(ctx, r, containerdStore, filesStore)
	if err != nil {
		return err
	}

	httpsSrv := &http.Server{
		Addr:      args.HttpsAddr,
		Handler:   handler,
		TLSConfig: r.Net().DefaultTLSConfig(),
	}
	g.Go(func() error {
		if err := httpsSrv.ListenAndServeTLS("", ""); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})
	g.Go(func() error {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return httpsSrv.Shutdown(shutdownCtx)
	})

	httpSrv := &http.Server{
		Addr:    args.HttpAddr,
		Handler: handler,
	}
	g.Go(func() error {
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})
	g.Go(func() error {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return httpSrv.Shutdown(shutdownCtx)
	})

	g.Go(func() error {
		http.Handle("/metrics/prometheus", promhttp.Handler())
		if err = http.ListenAndServe(args.PromAddr, nil); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	l.Info().Str("https", args.HttpsAddr).Str("http", args.HttpAddr).Str("router", args.RouterAddr).Str("prom", args.PromAddr).Msg("server start")
	err = g.Wait()
	if err != nil {
		return err
	}

	return nil
}

func addMirrorConfiguration(ctx context.Context, args *ServerCmd) error {
	if !args.AddMirrorConfiguration {
		return nil
	}

	l := zerolog.Ctx(ctx)
	fs := afero.NewOsFs()

	defaultHost, _ := url.Parse("https://mcr.microsoft.com")
	hosts := append([]url.URL{}, *defaultHost)

	var err error

	if len(args.Hosts) > 0 {
		hosts, err = toUrls(args.Hosts)
		if err != nil {
			return err
		}
	}

	l.Info().Msg(fmt.Sprintf("mirrors args: %v, hosts: %v", args.Hosts, hosts))

	defaultMirror, _ := url.Parse("https://localhost:30001")
	mirrors := append([]url.URL{}, *defaultMirror)

	if len(args.Mirrors) > 0 {
		mirrors, err = toUrls(args.Mirrors)
		if err != nil {
			return err
		}
	}

	err = containerd.AddHostsConfiguration(ctx, fs, args.ContainerdHostsConfigPath, hosts, mirrors, false)
	if err != nil {
		return err
	}

	return nil
}

func toUrls(hosts []string) ([]url.URL, error) {
	var urls []url.URL
	for _, h := range hosts {
		u, err := url.Parse(h)
		if err != nil {
			return nil, err
		}
		urls = append(urls, *u)
	}
	return urls, nil
}
