// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License, Version 2.0.
package random

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/azure/peerd/internal/math"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

var client = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // local kind clusters use self-signed certs
		},
	},
}

func Random(ctx context.Context, secrets string, n int, proxyHost string) error {
	l := zerolog.Ctx(ctx)

	if secrets == "" {
		return errors.New("secrets required")
	}

	if n <= 0 {
		return errors.New("node count must be positive")
	}

	if proxyHost == "" {
		return errors.New("proxy host required")
	}

	secretValue := strings.TrimSpace(secrets)

	// Parse the SAS URLs from the secret value
	upstreamSasUrls := strings.Fields(secretValue)
	p2pSasUrls := getP2pSasUrls(strings.Fields(secretValue), proxyHost)

	var g errgroup.Group
	var upstreamPercentiles, p2pPercentiles []float64
	var upstreamErrorRate, p2pErrorRate float64

	g.Go(func() error {
		upstreamPercentiles, upstreamErrorRate = benchmark(l, "upstream", upstreamSasUrls, n)
		return nil
	})

	g.Go(func() error {
		p2pPercentiles, p2pErrorRate = benchmark(l, "p2p", p2pSasUrls, n)
		return nil
	})

	_ = g.Wait()

	// Print the results
	if len(upstreamPercentiles) > 0 {
		l.Info().
			Float64("upstream.p50", upstreamPercentiles[0]).
			Float64("upstream.p75", upstreamPercentiles[1]).
			Float64("upstream.p90", upstreamPercentiles[2]).
			Float64("upstream.p95", upstreamPercentiles[3]).
			Float64("upstream.p100", upstreamPercentiles[4]).
			Msg("speeds (MB/s)")
	}

	if len(p2pPercentiles) > 0 {
		l.Info().
			Float64("p2p.p50", p2pPercentiles[0]).
			Float64("p2p.p75", p2pPercentiles[1]).
			Float64("p2p.p90", p2pPercentiles[2]).
			Float64("p2p.p95", p2pPercentiles[3]).
			Float64("p2p.p100", p2pPercentiles[4]).
			Msg("speeds (MB/s)")
	}

	l.Info().
		Float64("p2p.error_rate", p2pErrorRate).Float64("upstream.error_rate", upstreamErrorRate).
		Msg("error rates")

	return nil
}

// benchmark runs the benchmark and returns the measured download speeds.
func benchmark(l *zerolog.Logger, name string, urls []string, n int) ([]float64, float64) {
	log := l.With().Str("mode", name).Logger()

	// Group the SAS URLs randomly into groups of size n
	groups := math.RandomizedGroups(urls, n)

	// Download the SAS URLs in each group and measure the download speed
	var speeds []float64
	failures := 0

	var wg sync.WaitGroup
	for _, group := range groups {
		wg.Add(1)
		go func(group []string) {
			defer wg.Done()
			s, f := downloadSASURLs(&log, group)
			speeds = append(speeds, s...)
			failures += f
		}(group)
	}

	wg.Wait()

	// Calculate the percentiles
	return math.PercentilesFloat64Reverse(speeds, 0.50, 0.75, 0.9, 0.99, 1), float64(failures) / float64(len(urls))
}

// getP2pSasUrls returns the SAS URLs for the p2p network
func getP2pSasUrls(upstreamSasUrls []string, proxyHost string) []string {
	p2pSasUrls := make([]string, len(upstreamSasUrls))
	for _, s := range upstreamSasUrls {
		p2pSasUrls = append(p2pSasUrls, proxyHost+"/blobs/"+s)
	}
	return p2pSasUrls
}

// downloadSASURLs downloads the SAS URLs in a group and measures the download speed
func downloadSASURLs(l *zerolog.Logger, group []string) ([]float64, int) {
	readsPerBlob := 5
	var wg sync.WaitGroup
	l.Info().Int("groupSize", len(group)).Int("readsPerBlob", readsPerBlob).Strs("urls", group).Msg("downloading blobs")
	speeds := []float64{}
	failures := 0

	speedsChan := make(chan []float64, readsPerBlob)
	failuresChan := make(chan int, readsPerBlob)

	for _, sasURL := range group {
		wg.Add(1)
		go func(sasURL string) {
			defer wg.Done()

			if sasURL == "" {
				l.Warn().Str("url", sasURL).Msg("skipping SAS URL")
				return
			}

			blobSpeeds, f, err := downloadSASURL(l, sasURL, readsPerBlob)
			if err != nil {
				l.Error().Err(err).Str("url", sasURL).Msg("download error")
				failuresChan <- f
			}

			speedsChan <- blobSpeeds
		}(sasURL)
	}

	doneChan := make(chan bool, 1)
	go func() {
		wg.Wait()
		doneChan <- true
	}()

	for {
		select {
		case blobSpeeds := <-speedsChan:
			speeds = append(speeds, blobSpeeds...)

		case failure := <-failuresChan:
			failures += failure

		case <-doneChan:
			return speeds, failures
		}
	}
}

// downloadSASURL downloads a SAS URL and returns the number of bytes downloaded.
func downloadSASURL(l *zerolog.Logger, sasURL string, readsPerBlob int) ([]float64, int, error) {
	failures := -1
	req, err := http.NewRequest("HEAD", sasURL, nil)
	if err != nil {
		return nil, failures, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, failures, err
	} else if resp.StatusCode != http.StatusOK {
		return nil, failures, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}
	blobSize, _ := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)

	l.Info().Int64("size", blobSize).Int("readsPerBlob", readsPerBlob).Msg("downloading blob")

	errs := []error{}
	speeds := []float64{}
	failures = 0

	for i := 0; i < readsPerBlob; i++ {
		req, err = http.NewRequest("GET", sasURL, nil)
		if err != nil {
			return speeds, failures, err
		}

		// Set a random Range header
		s, _ := rand.Int(rand.Reader, big.NewInt(int64(blobSize)))
		start := s.Int64()
		e, _ := rand.Int(rand.Reader, big.NewInt(int64(blobSize-start+1)))
		end := start + e.Int64()
		contentRange := fmt.Sprintf("bytes=%d-%d", start, end)
		req.Header.Set("Range", contentRange)

		l2 := l.With().Int("size", int(blobSize)).Str("url", sasURL).Str("http.request.range", contentRange).Logger()

		st := time.Now()
		resp, err = client.Do(req)
		if err != nil {
			continue
		}
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
			l2.Error().Err(err).Int("status", resp.StatusCode).Msg("unexpected status code")
			errs = append(errs, fmt.Errorf("unexpected status code %d", resp.StatusCode))
			failures++
			continue
		}

		defer resp.Body.Close()
		n, err := io.Copy(io.Discard, resp.Body)
		if err != nil {
			l2.Error().Err(err).Msg("error reading response body")
			errs = append(errs, err)
			failures++
		} else {
			since := time.Since(st)
			speeds = append(speeds, float64(n)/since.Seconds())
		}

		sleep()
	}

	return speeds, failures, errors.Join(errs...)
}

func sleep() {
	var n int64
	_ = binary.Read(rand.Reader, binary.LittleEndian, &n)
	n = (n%(25-3+1) + 3)
	time.Sleep(time.Duration(n) * time.Millisecond)
}
