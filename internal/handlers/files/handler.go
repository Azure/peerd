// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package files

import (
	"context"
	"net/http"
	"os"
	"time"

	pcontext "github.com/azure/peerd/pkg/context"
	"github.com/azure/peerd/pkg/files/store"
	"github.com/azure/peerd/pkg/metrics"
)

// FilesHandler describes a handler for files.
type FilesHandler struct {
	store           store.FilesStore
	metricsRecorder metrics.Metrics
}

// Handle handles a request for a file.
func (h *FilesHandler) Handle(c pcontext.Context) {
	log := pcontext.Logger(c).With().Str("blob", pcontext.BlobUrl(c)).Bool("p2p", pcontext.IsRequestFromAPeer(c)).Logger()
	log.Debug().Msg("files handler start")
	s := time.Now()
	defer func() {
		dur := time.Since(s)
		h.metricsRecorder.RecordRequest(c.Request.Method, "files", float64(dur.Milliseconds()))
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
	w.Header().Set(pcontext.NodeHeaderKey, pcontext.NodeName)
	w.Header().Set(pcontext.CorrelationHeaderKey, c.GetString(pcontext.CorrelationIdCtxKey))

	http.ServeContent(w, c.Request, "file", time.Now(), f)
}

// fill fills the context with handler specific information.
func (h *FilesHandler) fill(c pcontext.Context) error {
	c.Set("handler", "files")

	key, d, err := h.store.Key(c)
	if err != nil {
		return err
	}

	c.Set(pcontext.DigestCtxKey, d.String())
	c.Set(pcontext.FileChunkCtxKey, key)
	c.Set(pcontext.BlobUrlCtxKey, pcontext.BlobUrl(c))
	c.Set(pcontext.BlobRangeCtxKey, c.Request.Header.Get("Range"))

	return nil
}

// New creates a new files handler.
func New(ctx context.Context, fs store.FilesStore) *FilesHandler {
	return &FilesHandler{fs, metrics.FromContext(ctx)}
}
