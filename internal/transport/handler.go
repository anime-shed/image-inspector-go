package transport

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"go-image-inspector/internal/analyzer"
	apperrors "go-image-inspector/internal/errors"
	"go-image-inspector/internal/logger"
	"go-image-inspector/internal/service"
	"go-image-inspector/pkg/config"
	"go-image-inspector/pkg/models"
	"go-image-inspector/pkg/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Remove package-level validator as validation is now handled by service layer

// AnalysisRequest is now an alias to the shared models.AnalysisRequest
type AnalysisRequest = models.AnalysisRequest

// AnalysisOptionsRequest is now an alias to the shared models.AnalysisOptionsRequest
type AnalysisOptionsRequest = models.AnalysisOptionsRequest

// ErrorResponse is now an alias to the shared models.ErrorResponse
type ErrorResponse = models.ErrorResponse

func NewHandler(analysisService service.ImageAnalysisService, detailedService *services.DetailedAnalysisService, cfg *config.Config) http.Handler {
	r := gin.Default()

	// Add middleware
	r.Use(
		requestSizeLimiter(cfg.MaxRequestBodySize),
		errorHandler(),
	)

	// Configure routes
	r.GET("/health", healthCheck)
	r.POST("/analyze", analyzeImage(analysisService, cfg))
	r.POST("/analyze/options", analyzeImageWithOptions(analysisService, cfg))
	if detailedService == nil {
		logger.WithField("route", "/detailed-analyze").
			Warn("DetailedAnalysisService is nil; route not registered")
	} else {
		if detailedService == nil {
			logger.WithField("route", "/detailed-analyze").
				Warn("DetailedAnalysisService is nil; route not registered")
		} else {
			r.POST("/detailed-analyze", detailedAnalyzeImage(detailedService, cfg))
		}
	}
	return r
}

func analyzeImage(analysisService service.ImageAnalysisService, cfg *config.Config) gin.HandlerFunc {
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

		// Check for IsOCR in query parameter (takes precedence over JSON body)
		if isOCRQuery := c.Query("IsOCR"); isOCRQuery != "" {
			req.IsOCR = isOCRQuery == "true"
		}

		// Log analysis attempt
		logger.WithFields(logrus.Fields{
			"url":    req.URL,
			"is_ocr": req.IsOCR,
		}).Debug("Starting image analysis")

		// Delegate business logic to service layer
		var response *service.ImageAnalysisResponse
		var err error

		if req.IsOCR && req.ExpectedText != "" {
			// Use OCR-specific analysis when expected text is provided
			response, err = analysisService.AnalyzeImageWithOCR(ctx, req.URL, req.ExpectedText)
		} else {
			// Use regular analysis
			response, err = analysisService.AnalyzeImage(ctx, req.URL, req.IsOCR)
		}

		if err != nil {
			// Log error with context
			logger.WithError(err).WithFields(logrus.Fields{
				"url": req.URL,
				"ip":  c.ClientIP(),
			}).Error("Image analysis failed")

			// Use custom error status code
			statusCode := apperrors.GetStatusCode(err)
			respondError(c, statusCode, "image analysis failed", err)
			return
		}

		// Log successful completion
		duration := time.Since(startTime)
		logger.WithFields(logrus.Fields{
			"url":                req.URL,
			"is_ocr":             req.IsOCR,
			"processing_time_ms": duration.Milliseconds(),
			"overexposed":        response.Quality.Overexposed,
			"oversaturated":      response.Quality.Oversaturated,
			"blurry":             response.Quality.Blurry,
		}).Info("Image analysis completed successfully")

		c.JSON(http.StatusOK, response)
	}
}

func analyzeImageWithOptions(analysisService service.ImageAnalysisService, cfg *config.Config) gin.HandlerFunc {
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
		}).Info("Processing image analysis request with options")

		var req AnalysisOptionsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			logger.WithError(err).WithFields(logrus.Fields{
				"ip": c.ClientIP(),
			}).Error("Invalid request format")
			respondError(c, http.StatusBadRequest, "invalid request format", err)
			return
		}

		// Use default options if none provided
		options := analyzer.DefaultOptions()
		if req.Options != nil {
			// Convert interface{} back to AnalysisOptions
			if opts, ok := req.Options.(*analyzer.AnalysisOptions); ok {
				options = *opts
			} else if optsMap, ok := req.Options.(map[string]interface{}); ok {
				// Handle JSON unmarshaling case where Options comes as map
				if ocrMode, exists := optsMap["ocr_mode"]; exists {
					if ocrModeBool, ok := ocrMode.(bool); ok {
						options.OCRMode = ocrModeBool
					}
				}
				if fastMode, exists := optsMap["fast_mode"]; exists {
					if fastModeBool, ok := fastMode.(bool); ok {
						options.FastMode = fastModeBool
					}
				}
				if qualityMode, exists := optsMap["quality_mode"]; exists {
					if qualityModeBool, ok := qualityMode.(bool); ok {
						options.QualityMode = qualityModeBool
					}
				}
			}
		}

		// Log analysis attempt
		logger.WithFields(logrus.Fields{
			"url":          req.URL,
			"ocr_mode":     options.OCRMode,
			"fast_mode":    options.FastMode,
			"quality_mode": options.QualityMode,
		}).Debug("Starting image analysis with options")

		// Delegate to service layer with options
		response, err := analysisService.AnalyzeImageWithOptions(ctx, req.URL, options)
		if err != nil {
			// Log error with context
			logger.WithError(err).WithFields(logrus.Fields{
				"url": req.URL,
				"ip":  c.ClientIP(),
			}).Error("Image analysis with options failed")

			// Use custom error status code
			statusCode := apperrors.GetStatusCode(err)
			respondError(c, statusCode, "image analysis failed", err)
			return
		}

		// Log successful completion
		duration := time.Since(startTime)
		logger.WithFields(logrus.Fields{
			"url":                req.URL,
			"ocr_mode":           options.OCRMode,
			"fast_mode":          options.FastMode,
			"quality_mode":       options.QualityMode,
			"processing_time_ms": duration.Milliseconds(),
			"overexposed":        response.Quality.Overexposed,
			"oversaturated":      response.Quality.Oversaturated,
			"blurry":             response.Quality.Blurry,
		}).Info("Image analysis with options completed successfully")

		c.JSON(http.StatusOK, response)
	}
}

func detailedAnalyzeImage(detailedService *services.DetailedAnalysisService, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// Log request start
		logger.WithFields(logrus.Fields{
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"user_agent": c.Request.UserAgent(),
			"ip":         c.ClientIP(),
		}).Info("Processing detailed image analysis request")

		var req AnalysisRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			logger.WithError(err).WithFields(logrus.Fields{
				"ip": c.ClientIP(),
			}).Error("Invalid request format")
			respondError(c, http.StatusBadRequest, "invalid request format", err)
			return
		}

		// Log analysis attempt
		logger.WithFields(logrus.Fields{
			"url": req.URL,
		}).Debug("Starting detailed image analysis")

		// Create detailed analysis request
		detailedReq := models.DetailedAnalysisRequest{
			URL: req.URL,
		}

		// Delegate to detailed service
		response, err := detailedService.AnalyzeImageDetailed(detailedReq)
		if err != nil {
			// Log error with context
			logger.WithError(err).WithFields(logrus.Fields{
				"url": req.URL,
				"ip":  c.ClientIP(),
			}).Error("Detailed image analysis failed")

			// Use custom error status code
			statusCode := apperrors.GetStatusCode(err)
			respondError(c, statusCode, "detailed image analysis failed", err)
			return
		}

		// Log successful completion
		duration := time.Since(startTime)
		logger.WithFields(logrus.Fields{
			"url":                req.URL,
			"processing_time_ms": duration.Milliseconds(),
		}).Info("Detailed image analysis completed successfully")

		c.JSON(http.StatusOK, response)
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
