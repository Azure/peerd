// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package oci

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	p2pcontext "github.com/azure/peerd/internal/context"
	"github.com/azure/peerd/internal/routing"
	"github.com/azure/peerd/pkg/peernet"
	"github.com/gin-gonic/gin"
)

var (
	// ResolveRetries is the number of times to attempt resolving a key before giving up.
	ResolveRetries = 3

	// ResolveTimeout is the timeout for resolving a key.
	ResolveTimeout = 1 * time.Second
)

// Mirror is a handler that handles requests to this registry Mirror.
type Mirror struct {
	resolveTimeout time.Duration
	router         routing.Router
	resolveRetries int

	n peernet.Network
}

var _ gin.HandlerFunc = (&Mirror{}).Handle

// Handle handles a request to this registry mirror.
func (m *Mirror) Handle(c *gin.Context) {
	key := c.GetString(p2pcontext.DigestCtxKey)
	if key == "" {
		key = c.GetString(p2pcontext.ReferenceCtxKey)
	}

	l := p2pcontext.Logger(c).With().Str("handler", "mirror").Str("ref", key).Logger()
	l.Debug().Msg("mirror handler start")
	s := time.Now()
	defer func() {
		l.Debug().Dur("duration", time.Since(s)).Msg("mirror handler stop")
	}()

	// Resolve mirror with the requested key
	resolveCtx, cancel := context.WithTimeout(c, m.resolveTimeout)
	defer cancel()

	if key == "" {
		// nolint
		c.AbortWithError(http.StatusInternalServerError, errors.New("neither digest nor reference provided"))
	}

	peersChan, err := m.router.Resolve(resolveCtx, key, false, m.resolveRetries)
	if err != nil {
		//nolint
		c.AbortWithError(http.StatusInternalServerError, err)
	}

	for {
		select {

		case <-resolveCtx.Done():
			// Resolving mirror has timed out.
			//nolint
			c.AbortWithError(http.StatusNotFound, fmt.Errorf(p2pcontext.PeerNotFoundLog))
			return

		case peer, ok := <-peersChan:
			// Channel closed means no more mirrors will be received and max retries has been reached.
			if !ok {
				//nolint
				c.AbortWithError(http.StatusInternalServerError, fmt.Errorf(p2pcontext.PeerResolutionExhaustedLog))
				return
			}

			succeeded := false
			u, err := url.Parse(peer.Addr)
			if err != nil {
				//nolint
				c.AbortWithError(http.StatusInternalServerError, err)
				return
			}

			proxy := httputil.NewSingleHostReverseProxy(u)
			proxy.Director = func(r *http.Request) {
				r.URL = u
				r.URL.Path = c.Request.URL.Path
				r.URL.RawQuery = c.Request.URL.RawQuery
				p2pcontext.SetOutboundHeaders(r, c)
			}
			proxy.ModifyResponse = func(resp *http.Response) error {
				if resp.StatusCode != http.StatusOK {
					return fmt.Errorf("expected peer to respond with 200, got: %s", resp.Status)
				}

				succeeded = true
				return nil
			}
			proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
				l.Error().Err(err).Msg("peer request failed, attempting next")
			}
			proxy.Transport = m.n.RoundTripperFor(peer.ID)

			proxy.ServeHTTP(c.Writer, c.Request)
			if !succeeded {
				break
			}

			l.Info().Str("peer", u.Host).Msg("request served from peer")
			return
		}
	}
}

// NewMirror creates a new mirror handler.
func NewMirror(router routing.Router) *Mirror {
	return &Mirror{
		resolveTimeout: ResolveTimeout,
		router:         router,
		resolveRetries: ResolveRetries,
		n:              router.Net(),
	}
}
