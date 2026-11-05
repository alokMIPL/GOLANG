package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	requestIDHeader = "X-Request-ID"
	requestIDKey    = "request_id"
)

var (
	Version = "dev"
	Commit  = "unknown"
)

var startedAt = time.Now()

type Pinger interface {
	Ping(ctx context.Context) error
}

type Config struct {
	DB           Pinger
	ReadyTimeout time.Duration
}

func NewRouter() *gin.Engine {
	return NewRouterWithConfig(Config{})
}

func NewRouterWithConfig(cfg Config) *gin.Engine {
	if cfg.ReadyTimeout <= 0 {
		cfg.ReadyTimeout = 2 * time.Second
	}

	r := gin.New()
	r.HandleMethodNotAllowed = true
	r.Use(requestID(), gin.Logger(), gin.Recovery(), securityHeaders())

	r.GET("/health", healthHandler)
	r.GET("/ready", readyHandler(cfg))
	r.GET("/version", versionHandler)

	v1 := r.Group("/api/v1")
	{
		v1.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "pong"})
		})
	}

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error":      "not found",
			"path":       c.Request.URL.Path,
			"request_id": c.GetString(requestIDKey),
		})
	})
	r.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, gin.H{
			"error":      "method not allowed",
			"method":     c.Request.Method,
			"request_id": c.GetString(requestIDKey),
		})
	})

	return r
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"ok":             true,
		"status":         "healthy",
		"uptime_seconds": int64(time.Since(startedAt).Seconds()),
	})
}

func readyHandler(cfg Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg.DB != nil {
			ctx, cancel := context.WithTimeout(c.Request.Context(), cfg.ReadyTimeout)
			defer cancel()
			if err := cfg.DB.Ping(ctx); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"ok":     false,
					"status": "not ready",
					"reason": "database unreachable",
				})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "status": "ready"})
	}
}

func versionHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version":    Version,
		"commit":     Commit,
		"started_at": startedAt.UTC().Format(time.RFC3339),
	})
}

func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(requestIDHeader)
		if id == "" || len(id) > 64 {
			id = newRequestID()
		}
		c.Set(requestIDKey, id)
		c.Header(requestIDHeader, id)
		c.Next()
	}
}

func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Cache-Control", "no-store")
		c.Next()
	}
}

func newRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b)
}
