// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License, Version 2.0.
package scanner

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/schollz/progressbar/v3"
)

const (
	path = "/usr/local/bin/scannerbase" // This should match the path in the Dockerfile.
)

func Scanner(ctx context.Context) error {
	l := zerolog.Ctx(ctx)

	sleep(l)

	l.Info().Str("path", path).Msg("starting scanner")

	f, _ := os.OpenFile(path, os.O_RDONLY, 0644)
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		l.Fatal().Err(err).Msg("failed to stat file")
		return err
	}

	size := info.Size()

	bar := progressbar.DefaultBytes(size, "reading")

	w, err := io.Copy(io.MultiWriter(io.Discard, bar), f)
	if err != nil {
		l.Fatal().Err(err).Msg("failed to read file")
		return err
	} else {
		l.Info().Int64("size", size).Int64("read", w).Msg("complete")
	}

	return nil
}

func sleep(l *zerolog.Logger) {
	var n uint64
	err := binary.Read(rand.Reader, binary.LittleEndian, &n)
	if err != nil {
		l.Error().Err(err).Msg("SLEEP FAILED")
		return
	}

	n = n % 100
	n = n + 1

	l.Info().Uint64("seconds", n).Msg("sleeping")

	// Sleep for n seconds
	time.Sleep(time.Duration(n) * time.Second)
}
