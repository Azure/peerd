// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package containerd

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	pcontext "github.com/azure/peerd/pkg/context"
	"github.com/azure/peerd/pkg/oci/distribution"
	"github.com/opencontainers/go-digest"
)

// Response headers
const (
	maxManifestSize           = 4 * 1024 * 1024
	dockerContentDigestHeader = "Docker-Content-Digest"
	contentLengthHeader       = "Content-Length"
	contentTypeHeader         = "Content-Type"
)

// Registry is a handler that handles requests to this registry.
type Registry struct {
	containerdStore Store
}

// Handle handles a request to this registry.
func (r *Registry) Handle(c pcontext.Context) {
	dgstStr := c.GetString(pcontext.DigestCtxKey)
	ref := c.GetString(pcontext.ReferenceCtxKey)
	var d digest.Digest
	var err error

	l := pcontext.Logger(c).With().Str("handler", "registry").Str("ref", ref).Str("digest", dgstStr).Logger()
	l.Debug().Msg("registry handler start")
	s := time.Now()
	defer func() {
		l.Debug().Dur("duration", time.Since(s)).Int("status", c.Writer.Status()).Str("digest", d.String()).Msg("registry handler stop")
	}()

	// Serve registry endpoints.
	if dgstStr == "" {
		d, err = r.containerdStore.Resolve(c, ref)
		if err != nil {
			//nolint
			c.AbortWithError(http.StatusNotFound, err)
			return
		}
	} else {
		d, err = digest.Parse(dgstStr)
		if err != nil {
			//nolint
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
	}

	refType, ok := c.Get(pcontext.RefTypeCtxKey)
	if !ok {
		//nolint
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("ref type not found in context"))
		return
	}

	switch refType.(distribution.ReferenceType) {

	case distribution.ReferenceTypeManifest:
		r.handleManifest(c, d)
		return

	case distribution.ReferenceTypeBlob:
		r.handleBlob(c, d)
		return
	}

	// If nothing matches return 404.
	c.Status(http.StatusNotFound)
}

// handleManifest handles a manifest request.
func (r *Registry) handleManifest(c pcontext.Context, dgst digest.Digest) {
	size, err := r.containerdStore.Size(c, dgst)
	if err != nil {
		//nolint
		c.AbortWithError(http.StatusNotFound, err)
		return
	} else if size >= maxManifestSize {
		//nolint
		c.AbortWithError(http.StatusNotFound, fmt.Errorf("refusing to serve a manifest larger than %v bytes, got: %v", maxManifestSize, size))
		return
	}

	b, mediaType, err := r.containerdStore.Bytes(c, dgst)
	if err != nil {
		//nolint
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.Header(contentTypeHeader, mediaType)
	c.Header(contentLengthHeader, strconv.FormatInt(int64(len(b)), 10))
	c.Header(dockerContentDigestHeader, dgst.String())

	if c.Request.Method == http.MethodHead {
		return
	}
	_, err = c.Writer.Write(b)
	if err != nil {
		//nolint
		c.AbortWithError(http.StatusServiceUnavailable, err)
		return
	}
}

// handleBlob handles a blob request.
func (r *Registry) handleBlob(c pcontext.Context, dgst digest.Digest) {
	size, err := r.containerdStore.Size(c, dgst)
	if err != nil {
		//nolint
		c.AbortWithError(http.StatusNotFound, err)
		return
	}

	c.Header(contentLengthHeader, strconv.FormatInt(size, 10))
	c.Header(dockerContentDigestHeader, dgst.String())
	if c.Request.Method == http.MethodHead {
		return
	}

	err = r.containerdStore.Write(c, c.Writer, dgst)
	if err != nil {
		//nolint
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
}

// NewRegistry creates a new registry handler.
func NewRegistry(containerdStore Store) *Registry {
	return &Registry{
		containerdStore: containerdStore,
	}
}
