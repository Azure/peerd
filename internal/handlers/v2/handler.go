// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package handlers

import (
	"context"
	"net/http"
	"path"
	"time"

	p2pcontext "github.com/azure/peerd/internal/context"
	"github.com/azure/peerd/internal/metrics"
	"github.com/azure/peerd/internal/oci"
	"github.com/azure/peerd/internal/oci/distribution"
	"github.com/azure/peerd/internal/routing"
	"github.com/azure/peerd/pkg/containerd"
	"github.com/gin-gonic/gin"
)

// V2Handler describes a handler for OCI content.
type V2Handler struct {
	mirror   *oci.Mirror
	registry *oci.Registry
}

var _ gin.HandlerFunc = (&V2Handler{}).Handle

// Handle handles a request for a file.
func (h *V2Handler) Handle(c *gin.Context) {
	l := p2pcontext.Logger(c).With().Bool("p2p", p2pcontext.IsRequestFromAPeer(c)).Logger()
	l.Debug().Msg("v2 handler start")
	s := time.Now()
	defer func() {
		dur := time.Since(s)
		metrics.Global.RecordRequest(c.Request.Method, "oci", dur.Seconds())
		l.Debug().Dur("duration", dur).Str("ns", c.GetString(p2pcontext.NamespaceCtxKey)).Str("ref", c.GetString(p2pcontext.ReferenceCtxKey)).Str("digest", c.GetString(p2pcontext.DigestCtxKey)).Msg("v2 handler stop")
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

	if p2pcontext.IsRequestFromAPeer(c) {
		h.registry.Handle(c)
		return
	} else {
		h.mirror.Handle(c)
		return
	}
}

// fill fills the context with handler specific information.
func (h *V2Handler) fill(c *gin.Context) error {
	c.Set("handler", "v2")

	ns := c.Query("ns")
	if ns == "" {
		ns = "docker.io"
	}

	c.Set(p2pcontext.NamespaceCtxKey, ns)

	ref, dgst, refType, err := distribution.ParsePathComponents(ns, c.Request.URL.Path)
	if err != nil {
		return err
	}

	c.Set(p2pcontext.ReferenceCtxKey, ref)
	c.Set(p2pcontext.DigestCtxKey, dgst.String())
	c.Set(p2pcontext.RefTypeCtxKey, refType)

	return nil
}

// New creates a new OCI content handler.
func New(ctx context.Context, router routing.Router, containerdStore containerd.Store) (*V2Handler, error) {
	return &V2Handler{
		mirror:   oci.NewMirror(router),
		registry: oci.NewRegistry(containerdStore),
	}, nil
}
