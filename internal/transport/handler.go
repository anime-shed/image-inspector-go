package transport

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"go-image-inspector/internal/analyzer"
	apperrors "go-image-inspector/internal/errors"
	"go-image-inspector/internal/logger"
	"go-image-inspector/internal/storage"
	"go-image-inspector/pkg/config"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func validateImageURL(imageURL string) error {
	// Parse URL
	parsedURL, err := url.Parse(imageURL)
	if err != nil {
		return apperrors.NewValidationError("Invalid URL format", err)
	}
	// Check if host is present
	if parsedURL.Host == "" {
		return apperrors.NewValidationError("URL must have a valid host", nil)
	}

	return nil
}

type AnalysisRequest struct {
	URL          string `json:"url" binding:"required,url"`
	IsOCR        bool   `json:"is_ocr,omitempty"`
	ExpectedText string `json:"expected_text,omitempty"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

func NewHandler(analyzer analyzer.ImageAnalyzer, fetcher storage.ImageFetcher, cfg *config.Config) http.Handler {
	r := gin.Default()

	// Add middleware
	r.Use(
		requestSizeLimiter(cfg.MaxRequestBodySize),
		errorHandler(),
	)

	// Configure routes
	r.GET("/health", healthCheck)
	r.POST("/analyze", analyzeImage(analyzer, fetcher, cfg))

	return r
}

func analyzeImage(a analyzer.ImageAnalyzer, f storage.ImageFetcher, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		ctx, cancel := context.WithTimeout(c.Request.Context(), cfg.RequestTimeout)
		defer cancel()

		// Log request start
		logger.WithFields(logrus.Fields{
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"user_agent": c.Request.UserAgent(),
			"ip":         c.ClientIP(),
		}).Info("Processing image analysis request")

		var req AnalysisRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			logger.WithError(err).WithFields(logrus.Fields{
				"ip": c.ClientIP(),
			}).Error("Invalid request format")
			respondError(c, http.StatusBadRequest, "invalid request format", err)
			return
		}

		// Validate image URL
		if err := validateImageURL(req.URL); err != nil {
			logger.WithError(err).WithFields(logrus.Fields{
				"url": req.URL,
				"ip":  c.ClientIP(),
			}).Error("Invalid image URL")

			// Use custom error status code
			statusCode := apperrors.GetStatusCode(err)
			respondError(c, statusCode, "invalid image URL", err)
			return
		}

		// Check for IsOCR in query parameter (takes precedence over JSON body)
		if isOCRQuery := c.Query("IsOCR"); isOCRQuery != "" {
			req.IsOCR = isOCRQuery == "true"
		}

		// Log image fetch attempt
		logger.WithFields(logrus.Fields{
			"url":    req.URL,
			"is_ocr": req.IsOCR,
		}).Debug("Fetching image")

		img, err := f.FetchImage(ctx, req.URL)
		if err != nil {
			// Wrap network/fetch errors with custom error type
			var fetchErr *apperrors.AppError
			if errors.Is(err, context.DeadlineExceeded) {
				fetchErr = apperrors.NewTimeoutError("Image fetch timeout", err)
			} else {
				fetchErr = apperrors.NewNetworkError("Failed to fetch image", err)
			}

			logger.WithError(fetchErr).WithFields(logrus.Fields{
				"url": req.URL,
				"ip":  c.ClientIP(),
			}).Error("Failed to fetch image")

			respondError(c, fetchErr.StatusCode, "failed to fetch image", fetchErr)
			return
		}

		var result analyzer.AnalysisResult
		if req.IsOCR {
			// Use OCR analysis when isOCR is true
			result = a.AnalyzeWithOCR(img, req.ExpectedText)
		} else {
			// Use regular analysis when isOCR is false
			result = a.Analyze(img, false)
		}

		// Log successful completion
		duration := time.Since(startTime)
		logger.WithFields(logrus.Fields{
			"url":                req.URL,
			"is_ocr":             req.IsOCR,
			"processing_time_ms": duration.Milliseconds(),
			"overexposed":        result.Overexposed,
			"oversaturated":      result.Oversaturated,
			"blurry":             result.Blurry,
		}).Info("Image analysis completed successfully")

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
	// Check if it's a custom app error first
	if appErr, ok := err.(*apperrors.AppError); ok {
		return appErr.StatusCode
	}

	// Fallback to context-based errors
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
	// Log the error with context
	logger.WithError(err).WithFields(logrus.Fields{
		"status_code": code,
		"message":     message,
		"path":        c.Request.URL.Path,
		"method":      c.Request.Method,
		"ip":          c.ClientIP(),
	}).Error("Request failed")

	c.AbortWithStatusJSON(code, ErrorResponse{
		Error:   http.StatusText(code),
		Message: fmt.Sprintf("%s: %v", message, err),
	})
}
