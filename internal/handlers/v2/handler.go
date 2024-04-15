// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package v2

import (
	"context"
	"net/http"
	"path"
	"time"

	"github.com/azure/peerd/pkg/containerd"
	pcontext "github.com/azure/peerd/pkg/context"
	"github.com/azure/peerd/pkg/discovery/routing"
	"github.com/azure/peerd/pkg/metrics"
	"github.com/azure/peerd/pkg/oci/distribution"
)

// V2Handler describes a handler for OCI content.
type V2Handler struct {
	mirror          *Mirror
	registry        *Registry
	metricsRecorder metrics.Metrics
}

// Handle handles a request for a file.
func (h *V2Handler) Handle(c pcontext.Context) {
	l := pcontext.Logger(c).With().Bool("p2p", pcontext.IsRequestFromAPeer(c)).Logger()
	l.Debug().Msg("v2 handler start")
	s := time.Now()
	defer func() {
		dur := time.Since(s)
		h.metricsRecorder.RecordRequest(c.Request.Method, "oci", dur.Seconds())
		l.Debug().Dur("duration", dur).Str("ns", c.GetString(pcontext.NamespaceCtxKey)).Str("ref", c.GetString(pcontext.ReferenceCtxKey)).Str("digest", c.GetString(pcontext.DigestCtxKey)).Msg("v2 handler stop")
	}()

	p := path.Clean(c.Request.URL.Path)
	if p == "/v2" || p == "/v2/" {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.Status(http.StatusNotFound)
			return
		}
		c.Status(http.StatusOK)
		return
	}

	err := h.fill(c)
	if err != nil {
		l.Debug().Err(err).Msg("failed to fill context")
		// nolint
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if pcontext.IsRequestFromAPeer(c) {
		h.registry.Handle(c)
		return
	} else {
		h.mirror.Handle(c)
		return
	}
}

// fill fills the context with handler specific information.
func (h *V2Handler) fill(c pcontext.Context) error {
	c.Set("handler", "v2")

	ns := c.Query("ns")
	if ns == "" {
		ns = "docker.io"
	}

	c.Set(pcontext.NamespaceCtxKey, ns)

	ref, dgst, refType, err := distribution.ParsePathComponents(ns, c.Request.URL.Path)
	if err != nil {
		return err
	}

	c.Set(pcontext.ReferenceCtxKey, ref)
	c.Set(pcontext.DigestCtxKey, dgst.String())
	c.Set(pcontext.RefTypeCtxKey, refType)

	return nil
}

// New creates a new OCI content handler.
func New(ctx context.Context, router routing.Router, containerdStore containerd.Store) (*V2Handler, error) {
	return &V2Handler{
		mirror:          NewMirror(router),
		registry:        NewRegistry(containerdStore),
		metricsRecorder: metrics.FromContext(ctx),
	}, nil
}
