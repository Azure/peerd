package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexflint/go-arg"
	p2pcontext "github.com/azure/peerd/internal/context"
	"github.com/azure/peerd/internal/files/store"
	"github.com/azure/peerd/internal/handlers"
	"github.com/azure/peerd/internal/k8s/events"
	"github.com/azure/peerd/internal/routing"
	"github.com/azure/peerd/internal/state"
	"github.com/azure/peerd/pkg/containerd"
	"github.com/rs/zerolog"
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

	l := zerolog.New(os.Stdout).With().Timestamp().Str("self", p2pcontext.NodeName).Str("version", version).Logger()
	ctx := l.WithContext(context.Background())

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

	ctx, err = events.WithContext(ctx)
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

	r, err := routing.NewRouter(ctx, args.RouterAddr, httpsPort)
	if err != nil {
		return err
	}

	containerdStore, err := containerd.NewDefaultStore([]string{"mcr.microsoft.com"})
	if err != nil {
		return err
	}
	err = containerdStore.Verify(ctx)
	if err != nil {
		return err
	}

	filesStore, err := store.NewFilesStore(ctx, r)
	if err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		state.Advertise(ctx, r, containerdStore, filesStore.Subscribe())
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
		return httpsSrv.Shutdown(shutdownCtx)
	})

	g.Go(func() error {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return httpSrv.Shutdown(shutdownCtx)
	})

	l.Info().Str("https", args.HttpsAddr).Str("http", args.HttpAddr).Msg("server start")
	err = g.Wait()
	if err != nil {
		return err
	}

	return nil
}
