// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/alexflint/go-arg"
	p2pcontext "github.com/azure/peerd/internal/context"
	"github.com/azure/peerd/tests/random"
	"github.com/azure/peerd/tests/scanner"
	"github.com/rs/zerolog"
)

func main() {
	args := &Arguments{}
	arg.MustParse(args)

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	l := zerolog.New(os.Stdout).With().Timestamp().Str("node", p2pcontext.NodeName).Str("version", version).Logger()
	ctx := l.WithContext(context.Background())

	err := run(ctx, args)
	if err != nil {
		l.Error().Err(err).Msg("error")
		os.Exit(1)
	}

	l.Info().Msg("shutdown")
}

func run(ctx context.Context, args *Arguments) error {
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGTERM)
	defer cancel()

	switch {
	case args.Version:
		zerolog.Ctx(ctx).Info().Msg("version") // version field is already added to the logger
		return nil

	case args.Random != nil:
		return random.Random(ctx, args.Random.Secrets, args.Random.NodeCount, args.Random.ProxyHost)

	case args.Scanner != nil:
		return scanner.Scanner(ctx)

	default:
		return fmt.Errorf("unknown subcommand")
	}
}
