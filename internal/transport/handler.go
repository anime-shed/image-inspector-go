package transport

import (
	"context"
	"errors"
	"fmt"
	"go-image-inspector/internal/analyzer"
	"go-image-inspector/internal/storage"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	maxRequestBodySize = 1 << 20 // 1MB
	requestTimeout     = 30 * time.Second
)

type AnalysisRequest struct {
	URL string `json:"url" binding:"required,url"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

func NewHandler(analyzer analyzer.ImageAnalyzer, fetcher storage.ImageFetcher) http.Handler {
	r := gin.Default()

	// Add middleware
	r.Use(
		requestSizeLimiter(maxRequestBodySize),
		errorHandler(),
	)

	// Configure routes
	r.GET("/health", healthCheck)
	r.POST("/analyze", analyzeImage(analyzer, fetcher))

	return r
}

func analyzeImage(a analyzer.ImageAnalyzer, f storage.ImageFetcher) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
		defer cancel()

		var req AnalysisRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			respondError(c, http.StatusBadRequest, "invalid request format", err)
			return
		}

		img, err := f.FetchImage(ctx, req.URL)
		if err != nil {
			respondError(c, http.StatusInternalServerError, "failed to fetch image", err)
			return
		}

		result := a.Analyze(img)
		c.JSON(http.StatusOK, result)
	}
}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "available",
		"version": "1.0.0",
		"time":    time.Now().UTC().Format(time.RFC3339),
	})
}

// Middleware and helper functions
func requestSizeLimiter(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

func errorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			respondError(c, determineStatusCode(err), "request processing failed", err)
		}
	}
}

func determineStatusCode(err error) int {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return http.StatusGatewayTimeout
	case errors.Is(err, context.Canceled):
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

func respondError(c *gin.Context, code int, message string, err error) {
	c.AbortWithStatusJSON(code, ErrorResponse{
		Error:   http.StatusText(code),
		Message: fmt.Sprintf("%s: %v", message, err),
	})
}
