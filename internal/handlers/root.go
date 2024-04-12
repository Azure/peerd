// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package handlers

import (
	"context"
	"net/http"
	"time"

	p2pcontext "github.com/azure/peerd/internal/context"
	filesStore "github.com/azure/peerd/internal/files/store"
	filesHandler "github.com/azure/peerd/internal/handlers/files"
	ociHandler "github.com/azure/peerd/internal/handlers/v2"
	"github.com/azure/peerd/pkg/containerd"
	"github.com/azure/peerd/pkg/discovery/routing"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

var fh *filesHandler.FilesHandler
var v2h *ociHandler.V2Handler

// Server creates a new HTTP server.
func Handler(ctx context.Context, r routing.Router, containerdStore containerd.Store, fs filesStore.FilesStore) (http.Handler, error) {
	var err error
	fh = filesHandler.New(ctx, fs)

	v2h, err = ociHandler.New(ctx, r, containerdStore)
	if err != nil {
		return nil, err
	}

	engine := newEngine(ctx)
	registerRoutes(engine, fileHandler, v2Handler)

	return engine, nil
}

// newEngine creates a new gin engine.
func newEngine(ctx context.Context) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()

	baseLog := zerolog.Ctx(ctx)

	engine.Use(func(c *gin.Context) {
		p2pcontext.FillCorrelationId(c)
		c.Set(p2pcontext.LoggerCtxKey, baseLog)

		l := p2pcontext.Logger(c)
		l.Debug().Msg("request start")
		s := time.Now()

		c.Next()

		status := c.Writer.Status()
		event := l.Info()
		if status >= 400 && status < 500 {
			event = l.Warn()
		} else if status >= 500 {
			event = l.Error()
		}

		if c.Errors != nil {
			errs := []error{}
			for _, e := range c.Errors {
				errs = append(errs, e.Err)
			}
			event = event.Errs("error", errs)
		}

		event.Dur("duration", time.Duration(time.Since(s).Seconds())).Str("method", c.Request.Method).Int("status", status).Msg("request served")
	})

	engine.Use(gin.Recovery())
	return engine
}

// registerRoutes registers the routes for the HTTP server.
func registerRoutes(engine *gin.Engine, f, v gin.HandlerFunc) {
	engine.HEAD("/blobs/*url", f)
	engine.GET("/blobs/*url", f)

	engine.HEAD("/v2", v)
	engine.GET("/v2", v)
	engine.HEAD("/v2/*ref", v)
	engine.GET("/v2/*ref", v)
}

// fileHandler is a handler function for the /blob API
// @Summary Get a blob by URL
// @Param url path string true "The URL of the blob"
// @Success 200 {string} string "The blob content"
// @Failure 404 {string} string "Not Found"
// @Router /blobs/{url} [get]
func fileHandler(c *gin.Context) {
	fh.Handle(c)
}

// v2Handler is a handler function for the /v2 API
// @Summary Get a manifest or a blob by repository and reference or digest
// @Param repo path string true "The repository name"
// @Param reference path string false "The reference of the manifest"
// @Param digest path string false "The digest of the blob"
// @Success 200 {object} map[string]string "The manifest or blob information"
// @Failure 404 {string} string "Not Found"
// @Router /v2/{repo}/manifests/{reference} [get]
// @Router /v2/{repo}/blobs/{digest} [get]
func v2Handler(c *gin.Context) {
	v2h.Handle(c)
}
