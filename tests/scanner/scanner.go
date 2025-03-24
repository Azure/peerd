// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package scanner

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	pcontext "github.com/azure/peerd/pkg/context"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/rs/zerolog"
	"github.com/schollz/progressbar/v3"
)

const (
	path = "/usr/local/bin/scannerbase" // This should match the path in the Dockerfile.
)

func Scanner(ctx context.Context) error {
	l := zerolog.Ctx(ctx)

	byteThroughputHist := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "scanner_bytes_throughput_mib_per_second",
		Help:    "Speed of scan in Mib per second.",
		Buckets: prometheus.ExponentialBuckets(1, 1.1, 100),
	}, []string{"self", "op"})
	prometheus.DefaultRegisterer.MustRegister(byteThroughputHist)

	errChan := make(chan error, 1)

	go func() {
		http.Handle("/metrics/prometheus", promhttp.Handler())
		if err := http.ListenAndServe("0.0.0.0:5004", nil); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
	}()

	sleep(l)

	l.Info().Str("path", path).Msg("starting scanner")

	f, _ := os.OpenFile(path, os.O_RDONLY, 0644)
	defer func() {
		if err := f.Close(); err != nil {
			l.Error().Err(err).Msg("failed to close file")
		}
	}()

	info, err := f.Stat()
	if err != nil {
		l.Fatal().Err(err).Msg("failed to stat file")
		return err
	}

	size := info.Size()

	bar := progressbar.DefaultBytes(size, "reading")
	measure(bar, time.Now(), l, byteThroughputHist, float64(size))
	w, err := io.Copy(io.MultiWriter(io.Discard, bar), f)
	if err != nil {
		l.Fatal().Err(err).Msg("failed to read file")
		return err
	} else {
		l.Info().Int64("size", size).Int64("read", w).Msg("complete")
	}

	if err := <-errChan; err != nil {
		l.Error().Msg(fmt.Sprintf("prom error: %v", err))
	}

	l.Info().Msg("sleeping to allow metrics scraping")
	time.Sleep(24 * 365 * time.Hour)

	return err
}

func measure(bar *progressbar.ProgressBar, startTime time.Time, l *zerolog.Logger, byteThroughputHist *prometheus.HistogramVec, size float64) {
	go func() {
		for {
			time.Sleep(500 * time.Microsecond)
			count := bar.State().CurrentBytes

			if count >= size {
				break
			}

			elapsed := time.Since(startTime)
			speed := float64(count) / elapsed.Seconds()
			l.Debug().Float64("speed", speed).Float64("bytes", count).Dur("elapsed", elapsed).Msg("speed")
			byteThroughputHist.WithLabelValues(pcontext.NodeName, "read").Observe(speed / float64(1024*1024))
		}
	}()
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
