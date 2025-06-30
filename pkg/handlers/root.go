// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package handlers

import (
	"context"
	"net/http"
	"time"

	pcontext "github.com/azure/peerd/pkg/context"
	"github.com/azure/peerd/pkg/discovery/routing"
	filesStore "github.com/azure/peerd/pkg/files/store"
	"github.com/azure/peerd/pkg/handlers/files"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

var fh *files.FilesHandler

// Server creates a new HTTP server.
func Handler(ctx context.Context, r routing.Router, fs filesStore.FilesStore) (http.Handler, error) {
	fh = files.New(ctx, fs)

	engine := newEngine(ctx)
	registerRoutes(engine, fileHandler)

	return engine, nil
}

// newEngine creates a new gin engine.
func newEngine(ctx context.Context) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()

	baseLog := zerolog.Ctx(ctx)

	engine.Use(func(c *gin.Context) {

		pc := pcontext.FromContext(c)

		pcontext.FillCorrelationId(pc)
		c.Set(pcontext.LoggerCtxKey, baseLog)

		l := pcontext.Logger(pc)
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
func registerRoutes(engine *gin.Engine, f gin.HandlerFunc) {
	engine.HEAD("/blobs/*url", f)
	engine.GET("/blobs/*url", f)
}

// fileHandler is a handler function for the /blob API
// @Summary Get a blob by URL
// @Param url path string true "The URL of the blob"
// @Success 200 {string} string "The blob content"
// @Failure 404 {string} string "Not Found"
// @Router /blobs/{url} [get]
func fileHandler(c *gin.Context) {
	fh.Handle(pcontext.FromContext(c))
}
