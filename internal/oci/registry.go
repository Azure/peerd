package oci

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	p2pcontext "github.com/azure/peerd/internal/context"
	"github.com/azure/peerd/internal/oci/distribution"
	"github.com/azure/peerd/pkg/containerd"
	"github.com/gin-gonic/gin"
	"github.com/opencontainers/go-digest"
	"github.com/rs/zerolog"
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
	containerdStore containerd.Store
}

var _ gin.HandlerFunc = (&Registry{}).Handle

// Handle handles a request to this registry.
func (r *Registry) Handle(c *gin.Context) {
	dgstStr := c.GetString(p2pcontext.DigestCtxKey)
	ref := c.GetString(p2pcontext.ReferenceCtxKey)
	var d digest.Digest
	var err error

	l := p2pcontext.Logger(c).With().Str("handler", "registry").Str("ref", ref).Str("digest", dgstStr).Logger()
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

	refType, ok := c.Get(p2pcontext.RefTypeCtxKey)
	if !ok {
		//nolint
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("ref type not found in context"))
	}

	switch refType.(distribution.ReferenceType) {

	case distribution.ReferenceTypeManifest:
		r.handleManifest(c, l, d)
		return

	case distribution.ReferenceTypeBlob:
		r.handleBlob(c, l, d)
		return
	}

	// If nothing matches return 404.
	c.Status(http.StatusNotFound)
}

// handleManifest handles a manifest request.
func (r *Registry) handleManifest(c *gin.Context, l zerolog.Logger, dgst digest.Digest) {
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
		c.AbortWithError(http.StatusNotFound, err)
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
		c.AbortWithError(http.StatusNotFound, err)
		return
	}
}

// handleBlob handles a blob request.
func (r *Registry) handleBlob(c *gin.Context, l zerolog.Logger, dgst digest.Digest) {
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
func NewRegistry(containerdStore containerd.Store) *Registry {
	return &Registry{
		containerdStore: containerdStore,
	}
}
