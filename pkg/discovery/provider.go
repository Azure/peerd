// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package discovery

import (
	"context"
	"errors"
	"fmt"
	"time"

	p2pcontext "github.com/azure/peerd/internal/context"
	"github.com/azure/peerd/pkg/containerd"
	"github.com/azure/peerd/pkg/discovery/routing"
	"github.com/rs/zerolog"
)

// Provide provides content on this host to peers on the network.
// It listens for events from the containerd.Store and filesChan channel to trigger the advertisement.
// The function runs until the context is done or an error occurs.
//
// Parameters:
// - ctx: The context.Context used for cancellation and deadline propagation.
// - r: The routing.Router used for advertising files.
// - containerdStore: The containerd.Store used for subscribing to events and advertising images.
// - filesChan: The channel that provides the files to be advertised.
//
// Returns: None.
func Provide(ctx context.Context, r routing.Router, containerdStore containerd.Store, filesChan <-chan string) {
	l := zerolog.Ctx(ctx).With().Str("component", "state").Logger()
	l.Debug().Msg("advertising start")
	s := time.Now()
	defer func() {
		l.Debug().Dur("duration", time.Since(s)).Msg("advertising stop")
	}()

	eventCh, errCh := containerdStore.Subscribe(ctx)

	immediate := make(chan time.Time, 1)
	immediate <- time.Now()

	expirationTicker := time.NewTicker(routing.MaxRecordAge - time.Minute)
	defer expirationTicker.Stop()

	ticker := p2pcontext.Merge(immediate, expirationTicker.C)

	for {
		select {

		case <-ctx.Done():
			return

		case <-ticker:
			l.Info().Msg("scheduled advertisement")
			err := provideAll(ctx, l, containerdStore, r)
			if err != nil {
				l.Error().Err(err).Msg("schedule: error advertising")
				continue
			}

		case ref := <-eventCh:
			l.Debug().Str("image", ref.Name()).Str("digest", ref.Digest().String()).Msg("advertising image")
			_, err := provideRef(ctx, l, containerdStore, r, ref)
			if err != nil {
				l.Error().Err(err).Msg("image: advertising error")
				continue
			}

		case blob := <-filesChan:
			l.Debug().Str("blob", blob).Msg("advertising file")
			err := r.Provide(ctx, []string{blob})
			if err != nil {
				l.Error().Err(err).Str("blob", blob).Msg("file: advertising error")
				continue
			}

		case err := <-errCh:
			l.Error().Err(err).Msg("channel error")
			continue
		}
	}
}

// provideAll provides all references in the containerd store using the provided logger and router.
// It returns an error if any error occurs during the advertisement process.
func provideAll(ctx context.Context, l zerolog.Logger, containerdStore containerd.Store, router routing.Router) error {
	refs, err := containerdStore.List(ctx)
	if err != nil {
		return err
	}

	errs := []error{}
	for _, ref := range refs {
		_, err := provideRef(ctx, l, containerdStore, router, ref)
		if err != nil {
			errs = append(errs, err)
			continue
		}
	}

	return errors.Join(errs...)
}

// provideRef provides the given containerd reference by extracting its digest and tags,
// retrieving additional digests from the containerd store, and advertising all the keys to the router.
// It returns the number of keys advertised and any error encountered.
func provideRef(ctx context.Context, l zerolog.Logger, containerdStore containerd.Store, router routing.Router, ref containerd.Reference) (int, error) {
	keys := []string{}
	keys = append(keys, ref.Digest().String())
	if ref.Tag() != "" {
		keys = append(keys, ref.String())
	}

	dgsts, err := containerdStore.All(ctx, ref)
	if err != nil {
		l.Error().Err(err).Str("image", ref.Name()).Str("digest", ref.Digest().String()).Msg("could not get digests for image")
	} else {
		keys = append(keys, dgsts...)
	}

	err = router.Provide(ctx, keys)
	if err != nil {
		return 0, fmt.Errorf("could not advertise image %v: %w", ref, err)
	}

	return len(keys), nil
}
