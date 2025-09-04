package service

import (
	"context"
	"image"
	"strings"
	"go-image-inspector/internal/analyzer"
	apperrors "go-image-inspector/internal/errors"
	"go-image-inspector/internal/repository"
	"go-image-inspector/pkg/models"
)

// ImageAnalysisService defines the interface for both basic and detailed image analysis
type ImageAnalysisService interface {
	// Basic analysis methods
	AnalyzeImage(ctx context.Context, imageURL string, isOCR bool) (*models.ImageAnalysisResponse, error)
	AnalyzeImageWithOCR(ctx context.Context, imageURL string, expectedText string) (*models.ImageAnalysisResponse, error)
	AnalyzeImageWithOptions(ctx context.Context, imageURL string, options analyzer.AnalysisOptions) (*models.ImageAnalysisResponse, error)

	// Detailed analysis methods
	AnalyzeImageDetailed(ctx context.Context, request models.DetailedAnalysisRequest) (*models.DetailedAnalysisResponse, error)

	// Common validation
	ValidateImageURL(imageURL string) error
}

// imageAnalysisService implements ImageAnalysisService with single analyzer
type imageAnalysisService struct {
	imageRepo repository.ImageRepository
	analyzer  analyzer.ImageAnalyzer
}

// NewImageAnalysisService creates a new image analysis service
func NewImageAnalysisService(
	imageRepository repository.ImageRepository,
	imageAnalyzer analyzer.ImageAnalyzer,
) ImageAnalysisService {
	return &imageAnalysisService{
		imageRepo: imageRepository,
		analyzer:  imageAnalyzer,
	}
}

// AnalyzeImage performs basic image analysis (legacy method for backward compatibility)
func (s *imageAnalysisService) AnalyzeImage(ctx context.Context, imageURL string, isOCR bool) (*models.ImageAnalysisResponse, error) {
	options := analyzer.DefaultOptions()
	options.OCRMode = isOCR
	return s.AnalyzeImageWithOptions(ctx, imageURL, options)
}

// AnalyzeImageWithOCR performs OCR-specific image analysis (legacy method for backward compatibility)
func (s *imageAnalysisService) AnalyzeImageWithOCR(ctx context.Context, imageURL string, expectedText string) (*models.ImageAnalysisResponse, error) {
	options := analyzer.OCROptions()
	options.OCRExpectedText = expectedText
	return s.AnalyzeImageWithOptions(ctx, imageURL, options)
}

// AnalyzeImageWithOptions performs image analysis with flexible configuration
func (s *imageAnalysisService) AnalyzeImageWithOptions(ctx context.Context, imageURL string, options analyzer.AnalysisOptions) (*models.ImageAnalysisResponse, error) {
	// Validate URL
	if err := s.ValidateImageURL(imageURL); err != nil {
		return nil, apperrors.NewValidationError("invalid image URL", err)
	}

	// Fetch image
	img, err := s.imageRepo.FetchImage(ctx, imageURL)
	if err != nil {
		return nil, apperrors.NewNetworkError("failed to fetch image", err)
	}

	// Analyze image with options using single analyzer
	result := s.analyzer.AnalyzeWithOptions(img, options)

	// Convert to basic response
	response := s.convertToBasicResponse(imageURL, &result)

	return response, nil
}

// AnalyzeImageDetailed performs comprehensive image analysis with detailed metrics
func (s *imageAnalysisService) AnalyzeImageDetailed(ctx context.Context, request models.DetailedAnalysisRequest) (*models.DetailedAnalysisResponse, error) {
	// Validate URL
	if err := s.ValidateImageURL(request.URL); err != nil {
		return nil, apperrors.NewValidationError("invalid image URL", err)
	}

	// Fetch image
	img, err := s.imageRepo.FetchImage(ctx, request.URL)
	if err != nil {
		return nil, apperrors.NewNetworkError("failed to fetch image", err)
	}

	// Configure detailed analysis options
	options := s.createDetailedAnalysisOptions(request)

	// Analyze image with same analyzer but detailed options
	result := s.analyzer.AnalyzeWithOptions(img, options)

	// Convert to detailed response
	response := s.convertToDetailedResponse(request.URL, &result, img)

	return response, nil
}

// ValidateImageURL validates the image URL
func (s *imageAnalysisService) ValidateImageURL(imageURL string) error {
	return s.imageRepo.ValidateImageURL(imageURL)
}

// createDetailedAnalysisOptions creates analysis options for detailed analysis
func (s *imageAnalysisService) createDetailedAnalysisOptions(request models.DetailedAnalysisRequest) analyzer.AnalysisOptions {
	options := analyzer.DefaultOptions()

	// Enable comprehensive analysis
	options.QualityMode = true
	options.UseWorkerPool = true
	options.SkipQRDetection = false
	options.SkipWhiteBalance = false
	options.SkipContourDetection = false
	options.SkipEdgeDetection = false

	// Apply custom thresholds if provided
	if request.CustomThresholds != nil {
		if request.CustomThresholds.BlurThreshold != nil {
			options.BlurThreshold = *request.CustomThresholds.BlurThreshold
		}
		if request.CustomThresholds.OverexposureThreshold != nil {
			options.OverexposureThreshold = *request.CustomThresholds.OverexposureThreshold
		}
		if request.CustomThresholds.OversaturationThreshold != nil {
			options.OversaturationThreshold = *request.CustomThresholds.OversaturationThreshold
		}
	}

	return options
}

// convertToBasicResponse converts analyzer result to basic service response
func (s *imageAnalysisService) convertToBasicResponse(imageURL string, result *models.AnalysisResult) *models.ImageAnalysisResponse {
	response := &models.ImageAnalysisResponse{
		ImageURL:          imageURL,
		Timestamp:         result.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
		ProcessingTimeSec: result.ProcessingTimeSec,
		Quality: models.Quality{
			Overexposed:   result.Quality.Overexposed,
			Oversaturated: result.Quality.Oversaturated,
			IncorrectWB:   result.Quality.IncorrectWB,
			Blurry:        result.Quality.Blurry,
			// Use the IsValid value calculated by the analyzer (includes quality validation errors)
			IsValid: result.Quality.IsValid,
		},
		Metrics: models.ImageMetrics{
			LaplacianVar:   result.Metrics.LaplacianVar,
			AvgLuminance:   result.Metrics.AvgLuminance,
			AvgSaturation:  result.Metrics.AvgSaturation,
			ChannelBalance: result.Metrics.ChannelBalance,
		},
		Errors: result.Errors,
	}

	// Add OCR result if available
	if result.OCRResult != nil {
		response.OCRResult = &models.OCRResult{
			ExtractedText: result.OCRResult.ExtractedText,
			ExpectedText:  result.OCRResult.ExpectedText,
			Confidence:    result.OCRResult.Confidence,
			MatchScore:    result.OCRResult.MatchScore,
		}
	}

	return response
}

// convertToDetailedResponse converts analyzer result to detailed service response
func (s *imageAnalysisService) convertToDetailedResponse(imageURL string, result *models.AnalysisResult, img interface{}) *models.DetailedAnalysisResponse {
	// Extract image dimensions
	width, height := s.getImageDimensions(img)

	// Get image metadata (content length, format, etc.)
	metadata, err := s.imageRepo.GetImageMetadata(context.Background(), imageURL)
	if err != nil {
		// Fallback to defaults if metadata fetch fails
		metadata = &models.ImageMetadata{
			ContentType:   "image/jpeg",
			ContentLength: 0,
			Format:        "JPEG",
		}
	}

	response := &models.DetailedAnalysisResponse{
		ImageURL:          imageURL,
		Timestamp:         result.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
		ProcessingTimeSec: result.ProcessingTimeSec,
		ImageMetadata: models.ImageMetadata{
			Width:         width,
			Height:        height,
			Format:        strings.ToLower(metadata.Format),
			ContentType:   metadata.ContentType,
			ContentLength: metadata.ContentLength,
		},
		QualityAnalysis: models.QualityAnalysis{
			Blurry:              result.Quality.Blurry,
			Overexposed:         result.Quality.Overexposed,
			Oversaturated:       result.Quality.Oversaturated,
			IncorrectWB:         result.Quality.IncorrectWB,
			IsLowResolution:     result.Quality.IsLowResolution,
			IsTooDark:           result.Quality.IsTooDark,
			IsTooBright:         result.Quality.IsTooBright,
			IsSkewed:            result.Quality.IsSkewed,
			HasDocumentEdges:    result.Quality.HasDocumentEdges,
			QRDetected:          result.Quality.QRDetected,
			IsValid:             result.Quality.IsValid,
			IsOCRReady:          s.calculateOCRReadiness(result),
			HasCriticalIssues:   s.hasCriticalIssues(result),
			OverallQualityScore: s.calculateQualityScore(result),
			SharpnessScore:      s.calculateSharpnessScore(result),
			ExposureScore:       s.calculateExposureScore(result),
			ColorScore:          s.calculateColorScore(result),
		},
		RawMetrics: models.RawMetrics{
			LaplacianVariance: result.Metrics.LaplacianVar,
			AvgLuminance:      result.Metrics.AvgLuminance,
			AvgSaturation:     result.Metrics.AvgSaturation,
			ChannelBalance:    result.Metrics.ChannelBalance,
			Brightness:        result.Metrics.AvgLuminance * 255, // Convert to 0-255 scale
			TotalPixels:       width * height,
		},
		Thresholds: models.AppliedThresholds{
			MinLaplacianVariance:    100.0,
			OverexposureThreshold:   0.95,
			OversaturationThreshold: 0.9,
			MinBrightness:           80.0,
			MaxBrightness:           220.0,
		},
		QualityChecks:     s.generateQualityChecks(result),
		OverallAssessment: s.generateOverallAssessment(result),
		ProcessingDetails: models.ProcessingDetails{
			FeaturesAnalyzed:   []string{"sharpness", "exposure", "color", "resolution"},
			ProcessingOptions:  map[string]interface{}{"analysis_mode": "comprehensive"},
			PerformanceMetrics: models.PerformanceMetrics{},
		},
		Errors: result.Errors,
	}

	// Add OCR analysis if available
	if result.OCRResult != nil {
		response.OCRAnalysis = &models.DetailedOCRAnalysis{
			OCRReadinessScore:  85.0, // Could be calculated based on quality metrics
			TextDetectionScore: 90.0,
			DocumentType:       "text",
			TextDensity:        0.3,
			EstimatedTextLines: 10,
		}
	}

	return response
}

// Helper methods
func (s *imageAnalysisService) getImageDimensions(img interface{}) (int, int) {
	// Type assertion to get image dimensions from standard Go image.Image
	if image, ok := img.(image.Image); ok {
		bounds := image.Bounds()
		return bounds.Dx(), bounds.Dy()
	}
	return 0, 0 // Default fallback
}

func (s *imageAnalysisService) calculateQualityScore(result *models.AnalysisResult) float64 {
	score := 100.0

	if result.Quality.Blurry {
		score -= 30.0
	}
	if result.Quality.Overexposed {
		score -= 25.0
	}
	if result.Quality.Oversaturated {
		score -= 20.0
	}
	if result.Quality.IncorrectWB {
		score -= 15.0
	}

	if score < 0 {
		score = 0
	}

	return score
}

func (s *imageAnalysisService) calculateSharpnessScore(result *models.AnalysisResult) float64 {
	if result.Quality.Blurry {
		return 0.0
	}
	// Convert Laplacian variance to a 0-100 score
	// Higher variance = sharper image
	if result.Metrics.LaplacianVar > 1000 {
		return 100.0
	} else if result.Metrics.LaplacianVar > 500 {
		return 80.0
	} else if result.Metrics.LaplacianVar > 100 {
		return 60.0
	} else {
		return 30.0
	}
}

func (s *imageAnalysisService) calculateExposureScore(result *models.AnalysisResult) float64 {
	if result.Quality.Overexposed || result.Quality.IsTooBright {
		return 20.0
	}
	if result.Quality.IsTooDark {
		return 30.0
	}
	// Good exposure range
	if result.Metrics.AvgLuminance >= 0.3 && result.Metrics.AvgLuminance <= 0.7 {
		return 100.0
	} else if result.Metrics.AvgLuminance >= 0.2 && result.Metrics.AvgLuminance <= 0.8 {
		return 80.0
	} else {
		return 60.0
	}
}

func (s *imageAnalysisService) calculateColorScore(result *models.AnalysisResult) float64 {
	if result.Quality.Oversaturated {
		return 40.0
	}
	if result.Quality.IncorrectWB {
		return 50.0
	}
	// Good saturation range
	if result.Metrics.AvgSaturation >= 0.2 && result.Metrics.AvgSaturation <= 0.8 {
		return 100.0
	} else if result.Metrics.AvgSaturation >= 0.1 && result.Metrics.AvgSaturation <= 0.9 {
		return 80.0
	} else {
		return 60.0
	}
}

func (s *imageAnalysisService) calculateOCRReadiness(result *models.AnalysisResult) bool {
	// OCR ready if not blurry, not too dark/bright, not skewed, and has good resolution
	return !result.Quality.Blurry && 
		!result.Quality.IsTooDark && 
		!result.Quality.IsTooBright && 
		!result.Quality.IsSkewed && 
		!result.Quality.IsLowResolution
}

func (s *imageAnalysisService) hasCriticalIssues(result *models.AnalysisResult) bool {
	// Critical issues that make image unusable
	return result.Quality.Blurry || 
		result.Quality.Overexposed || 
		result.Quality.IsLowResolution
}

func (s *imageAnalysisService) generateQualityChecks(result *models.AnalysisResult) []models.QualityCheckResult {
	checks := []models.QualityCheckResult{}

	checks = append(checks, models.QualityCheckResult{
		CheckName:      "Blur Detection",
		Passed:         !result.Quality.Blurry,
		ActualValue:    result.Metrics.LaplacianVar,
		ThresholdValue: 100.0,
		Message:        "Image sharpness assessment",
		Severity:       "error",
		Confidence:     0.85,
	})

	checks = append(checks, models.QualityCheckResult{
		CheckName:      "Exposure Check",
		Passed:         !result.Quality.Overexposed,
		ActualValue:    result.Metrics.AvgLuminance,
		ThresholdValue: 0.95,
		Message:        "Image exposure assessment",
		Severity:       "error",
		Confidence:     0.90,
	})

	checks = append(checks, models.QualityCheckResult{
		CheckName:      "Saturation Check",
		Passed:         !result.Quality.Oversaturated,
		ActualValue:    result.Metrics.AvgSaturation,
		ThresholdValue: 0.9,
		Message:        "Image saturation assessment",
		Severity:       "warning",
		Confidence:     0.80,
	})

	return checks
}

func (s *imageAnalysisService) generateOverallAssessment(result *models.AnalysisResult) models.OverallAssessment {
	qualityScore := s.calculateQualityScore(result)

	grade := "A"
	if qualityScore < 50 {
		grade = "F"
	} else if qualityScore < 70 {
		grade = "D"
	} else if qualityScore < 80 {
		grade = "C"
	} else if qualityScore < 90 {
		grade = "B"
	}

	return models.OverallAssessment{
		QualityGrade:   grade,
		UsabilityScore: qualityScore,
		SuitableFor:    []string{"web", "display"},
	}
}
