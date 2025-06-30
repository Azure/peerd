// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

// Package provider implements content advertisement for the peerd P2P system,
// advertising available content to peers via DHT routing.
package provider

import (
	"context"
	"time"

	"github.com/azure/peerd/pkg/discovery/routing"
	"github.com/rs/zerolog"
)

// Provide advertises content availability to the DHT network.
// Runs in a blocking loop, listening for blob identifiers on filesChan and
// advertising them via the routing system. Exits when ctx is cancelled.
//
// Parameters:
//   - ctx: Context for cancellation and deadline propagation
//   - r: Router for advertising content through the DHT
//   - filesChan: Channel receiving blob identifiers (SHA256 digest, optionally
//     with range suffix) to advertise
func Provide(ctx context.Context, r routing.Router, filesChan <-chan string) {
	l := zerolog.Ctx(ctx).With().Str("component", "state").Logger()
	l.Debug().Msg("advertising start")
	s := time.Now()
	defer func() {
		l.Debug().Dur("duration", time.Since(s)).Msg("advertising stop")
	}()

	immediate := make(chan time.Time, 1)
	immediate <- time.Now()

	for {
		select {

		case <-ctx.Done():
			return

		case blob := <-filesChan:
			l.Debug().Str("blob", blob).Msg("advertising file")
			err := r.Provide(ctx, []string{blob})
			if err != nil {
				l.Error().Err(err).Str("blob", blob).Msg("file: advertising error")
				continue
			}
		}
	}
}
