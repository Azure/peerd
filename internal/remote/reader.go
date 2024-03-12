// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package remote

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	p2pcontext "github.com/azure/peerd/internal/context"
	"github.com/azure/peerd/internal/metrics"
	"github.com/azure/peerd/internal/routing"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

type operation int

const (
	operationFstatRemote = operation(iota)
	operationPreadRemote
)

var errPeerNotFound = errors.New("peer not found")

// reader is a Reader implementation.
type reader struct {
	context        *gin.Context
	resolveTimeout time.Duration

	router            routing.Router
	resolveRetries    int
	defaultHttpClient *http.Client
}

var _ Reader = &reader{}

// Log returns the logger with context for this reader.
func (r *reader) Log() *zerolog.Logger {
	l := p2pcontext.Logger(r.context)
	return &l
}

// PreadRemote is like pread but to a remote file.
func (r *reader) PreadRemote(buf []byte, offset int64) (int, error) {
	key := r.context.GetString(p2pcontext.FileChunkCtxKey)
	start := offset
	end := int64(len(buf)) + offset - 1

	log := r.Log().With().Str("operation", "preadremote").Str("key", key).Int64("start", start).Int64("end", end).Logger()

	count, err := r.doP2p(log, key, start, end, operationPreadRemote, buf)
	if err == nil {
		return int(count), nil
	}

	// Could not find a peer that has this file, request origin.
	startTime := time.Now()
	originReq, err := r.originRequest(start, end)
	if err != nil {
		return -1, err
	}

	count32 := int(0)
	defer func() {
		metrics.Global.RecordUpstreamResponse(originReq.URL.Hostname(), key, "pread", time.Since(startTime).Seconds(), int64(count32))
	}()
	count32, err = r.preadRemote(log, originReq, r.defaultHttpClient, buf)
	return count32, err
}

// FstatRemote stats a remote file.
func (r *reader) FstatRemote() (int64, error) {
	key := r.context.GetString(p2pcontext.FileChunkCtxKey)
	start := int64(0)
	end := int64(0)

	log := r.Log().With().Str("operation", "fstatremote").Int64("start", start).Int64("end", end).Str("key", key).Logger()

	startTime := time.Now()
	originReq, err := r.originRequest(start, end)
	if err != nil {
		return -1, err
	}

	var count int64
	defer func() {
		metrics.Global.RecordUpstreamResponse(originReq.URL.Hostname(), key, "fstat", time.Since(startTime).Seconds(), count)
	}()
	count, err = r.fstatRemote(log, originReq, r.defaultHttpClient)
	return count, err
}

// doP2p tries to resolve the key in the p2p network and if successful, it will perform the operation on the peer, and return the result.
func (r *reader) doP2p(log zerolog.Logger, fileChunkKey string, start, end int64, o operation, buf []byte) (int64, error) {
	if p2pcontext.IsRequestFromAPeer(r.context) {
		log.Warn().Msg("refusing to propagate request from one peer to another")
		return -1, errPeerNotFound
	}

	log.Debug().Msg(p2pcontext.PeerResolutionStartLog)
	defer log.Debug().Msg(p2pcontext.PeerResolutionStopLog)

	resolveCtx, cancel := context.WithTimeout(log.WithContext(r.context), r.resolveTimeout)
	defer cancel()

	startTime := time.Now()
	peerCount := 0
	peersCh, negCacheCallback, err := r.router.ResolveWithCache(resolveCtx, fileChunkKey, false, r.resolveRetries)
	if err != nil {
		//nolint:errcheck // ignore
		log.Error().Err(err).Msg(p2pcontext.PeerRequestErrorLog)
		return -1, err
	}

	// Request a peer for this file.
peerLoop:
	for {
		select {

		case <-resolveCtx.Done():
			// Resolving mirror has timed out.
			negCacheCallback()
			log.Info().Msg(p2pcontext.PeerNotFoundLog)
			break peerLoop

		case peer, ok := <-peersCh:
			// Channel closed means no more mirrors will be received and max retries has been reached.
			if !ok {
				negCacheCallback()
				log.Info().Msg(p2pcontext.PeerResolutionExhaustedLog)
				break peerLoop
			}

			if peerCount == 0 {
				// Only report the time it took to discover the first peer.
				metrics.Global.RecordPeerDiscovery(peer.Addr, time.Since(startTime).Seconds())
				peerCount++
			}

			peerReq, err := r.peerRequest(peer.Addr, start, end)
			if err != nil {
				log.Error().Err(err).Msg(p2pcontext.PeerRequestErrorLog)
				// try next peer
				break
			}

			client := r.router.Net().HTTPClientFor(peer.ID)

			var count int64
			startTime = time.Now()
			if o == operationFstatRemote {
				count, err = r.fstatRemote(log, peerReq, client)
			} else if o == operationPreadRemote {
				var c int
				c, err = r.preadRemote(log, peerReq, client, buf)
				count = int64(c)
			} else {
				err = fmt.Errorf("unknown operation: %v", o)
			}

			if err != nil {
				// try next peer
				log.Error().Err(err).Msg(p2pcontext.PeerRequestErrorLog)
			} else {
				op := "fstat"
				if o == operationPreadRemote {
					op = "pread"
				}
				metrics.Global.RecordPeerResponse(peer.Addr, fileChunkKey, op, time.Since(startTime).Seconds(), count)
				return count, nil
			}
		}
	}

	return -1, errPeerNotFound
}

// fstatRemote stats the file.
func (r *reader) fstatRemote(log zerolog.Logger, req *http.Request, client *http.Client) (int64, error) {
	log.Debug().Str("url", req.URL.String()).Str("range", req.Header.Get("Range")).Msg("reader fstatRemote start")
	defer log.Debug().Msg("reader fstatRemote stop")

	resp, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("reader fstatRemote error")
		return 0, Error{resp, err}
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return resp.ContentLength, nil
	}

	if resp.StatusCode == 206 {
		l := resp.ContentLength
		rs := resp.Header.Get("Content-Range")
		if rs == "" {
			return l, nil
		}

		pos := strings.LastIndexByte(rs, '/')
		if pos < 0 {
			return l, nil
		}

		l, _ = strconv.ParseInt(rs[pos+1:], 10, 64)
		return l, nil
	}

	log.Error().Err(err).Int("status", resp.StatusCode).Msg("reader fstatRemote error")
	return 0, Error{resp, fmt.Errorf("unexpected response code: %d", resp.StatusCode)}
}

// preadRemote reads the file.
func (r *reader) preadRemote(log zerolog.Logger, req *http.Request, client *http.Client, buf []byte) (int, error) {
	log.Debug().Str("url", req.URL.String()).Str("range", req.Header.Get("Range")).Msg("reader preadRemote start")
	statusCode := -1
	s := time.Now()
	defer func() {
		log.Debug().Int("status", statusCode).Dur("duration", time.Since(s)).Msg("reader preadRemote stop")
	}()

	resp, err := client.Do(req)
	if resp != nil {
		statusCode = resp.StatusCode
	}
	if err != nil {
		detailedErr := Error{resp, err}
		log.Error().Err(detailedErr).Str("url", req.URL.String()).Str("range", req.Header.Get("Range")).Msg("reader preadRemote error")
		return 0, detailedErr
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 206 {
		log.Error().Err(err).Int("status", resp.StatusCode).Msg("reader preadRemote error")
		return 0, Error{resp, fmt.Errorf("unexpected response code: %d", resp.StatusCode)}
	}

	return io.ReadFull(resp.Body, buf)
}

// originRequest will create a new request to origin.
func (r *reader) originRequest(start, end int64) (*http.Request, error) {
	return r.remoteRequest(r.context.GetString(p2pcontext.BlobUrlCtxKey), start, end)
}

// perRequest will create a new request to a peer.
func (r *reader) peerRequest(peer string, start, end int64) (*http.Request, error) {
	return r.remoteRequest(fmt.Sprintf("%v/blobs/%v", peer, r.context.GetString(p2pcontext.BlobUrlCtxKey)), start, end)
}

// remoteRequest creates a new HTTP request to a remote server.
func (r *reader) remoteRequest(u string, start, end int64) (*http.Request, error) {
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	for key, vals := range r.context.Request.Header {
		vals2 := make([]string, len(vals))
		copy(vals2, vals)
		req.Header[key] = vals2
	}

	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
	p2pcontext.SetOutboundHeaders(req, r.context)

	return req, nil
}

// NewReader creates a new remote reader.
func NewReader(c *gin.Context, router routing.Router, resolveRetries int, resolveTimeout time.Duration) Reader {
	cc := c.Copy()
	return &reader{
		context:           cc,
		resolveTimeout:    resolveTimeout,
		router:            router,
		resolveRetries:    resolveRetries,
		defaultHttpClient: router.Net().HTTPClientFor(""),
	}
}
