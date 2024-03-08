package handlers

import (
	"context"
	"net/http"
	"os"
	"time"

	p2pcontext "github.com/azure/peerd/internal/context"
	"github.com/azure/peerd/internal/files/store"
	"github.com/azure/peerd/internal/metrics"
	"github.com/gin-gonic/gin"
)

// FilesHandler describes a handler for files.
type FilesHandler struct {
	store store.FilesStore
}

var _ gin.HandlerFunc = (&FilesHandler{}).Handle

// Handle handles a request for a file.
func (h *FilesHandler) Handle(c *gin.Context) {
	log := p2pcontext.Logger(c).With().Str("blob", p2pcontext.BlobUrl(c)).Bool("p2p", p2pcontext.IsRequestFromAPeer(c)).Logger()
	log.Debug().Msg("files handler start")
	s := time.Now()
	defer func() {
		dur := time.Since(s)
		metrics.Global.RecordRequest(c.Request.Method, "files", float64(dur.Milliseconds()))
		log.Debug().Dur("duration", dur).Msg("files handler stop")
	}()

	err := h.fill(c)
	if err != nil {
		log.Debug().Err(err).Msg("failed to fill context")
		// nolint
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	f, err := h.store.Open(c)
	if err == os.ErrNotExist {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	if err != nil {
		// nolint
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	w := c.Writer

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Del("Content-Length")
	w.Header().Set(p2pcontext.NodeHeaderKey, p2pcontext.NodeName)
	w.Header().Set(p2pcontext.CorrelationHeaderKey, c.GetString(p2pcontext.CorrelationIdCtxKey))

	http.ServeContent(w, c.Request, "file", time.Now(), f)
}

// fill fills the context with handler specific information.
func (h *FilesHandler) fill(c *gin.Context) error {
	c.Set("handler", "files")

	key, d, err := h.store.Key(c)
	if err != nil {
		return err
	}

	c.Set(p2pcontext.DigestCtxKey, d.String())
	c.Set(p2pcontext.FileChunkCtxKey, key)
	c.Set(p2pcontext.BlobUrlCtxKey, p2pcontext.BlobUrl(c))
	c.Set(p2pcontext.BlobRangeCtxKey, c.Request.Header.Get("Range"))

	return nil
}

// New creates a new files handler.
func New(ctx context.Context, fs store.FilesStore) *FilesHandler {
	return &FilesHandler{fs}
}
