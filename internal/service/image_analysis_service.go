package service

import (
	"context"
	"github.com/anime-shed/image-inspector-go/internal/analyzer"
	apperrors "github.com/anime-shed/image-inspector-go/internal/errors"
	"github.com/anime-shed/image-inspector-go/internal/repository"
	"github.com/anime-shed/image-inspector-go/pkg/models"
	"image"
	"strings"
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

	// Convert to detailed response with full context
	response := s.convertToDetailedResponse(ctx, request, options, &result, img)

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

	// Analysis mode
	switch strings.ToLower(strings.TrimSpace(request.AnalysisMode)) {
	case "ocr":
		o := analyzer.OCROptions()
		o.UseWorkerPool = options.UseWorkerPool
		options = o
		if request.ExpectedText != "" {
			options.OCRExpectedText = request.ExpectedText
		}
	}

	// Feature flags
	if request.FeatureFlags != nil {
		if v := request.FeatureFlags["skip_qr_detection"]; v {
			options.SkipQRDetection = true
		}
		if v := request.FeatureFlags["skip_white_balance"]; v {
			options.SkipWhiteBalance = true
		}
		if v := request.FeatureFlags["skip_contour_detection"]; v {
			options.SkipContourDetection = true
		}
		if v := request.FeatureFlags["skip_edge_detection"]; v {
			options.SkipEdgeDetection = true
		}
	}

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
func (s *imageAnalysisService) convertToDetailedResponse(ctx context.Context, request models.DetailedAnalysisRequest, options analyzer.AnalysisOptions, result *models.AnalysisResult, img image.Image) *models.DetailedAnalysisResponse {
	// Extract image dimensions
	width, height := s.getImageDimensions(img)

	// Get image metadata (content length, format, etc.)
	metadata, err := s.imageRepo.GetImageMetadata(ctx, request.URL)
	if err != nil {
		// Fallback to defaults if metadata fetch fails
		metadata = &models.ImageMetadata{
			ContentType:   "image/jpeg",
			ContentLength: 0,
			Format:        "JPEG",
		}
	}

	response := &models.DetailedAnalysisResponse{
		ImageURL:          request.URL,
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
			Overexposed:         result.Quality.Overexposed,
			Oversaturated:       result.Quality.Oversaturated,
			IncorrectWB:         result.Quality.IncorrectWB,
			Blurry:              result.Quality.Blurry,
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
		RawMetrics: func() models.RawMetrics {
			b := result.Metrics.AvgLuminance * 255.0
			if b < 0 {
				b = 0
			}
			if b > 255 {
				b = 255
			}
			return models.RawMetrics{
				LaplacianVariance: result.Metrics.LaplacianVar,
				AvgLuminance:      result.Metrics.AvgLuminance,
				AvgSaturation:     result.Metrics.AvgSaturation,
				ChannelBalance:    result.Metrics.ChannelBalance,
				Brightness:        b,
				Width:             width,
				Height:            height,
				TotalPixels:       width * height,
			}
		}(),
		Thresholds: func() models.AppliedThresholds {
			t := models.AppliedThresholds{
				MinLaplacianVariance:    options.BlurThreshold,
				OverexposureThreshold:   options.OverexposureThreshold,
				OversaturationThreshold: options.OversaturationThreshold,
				MinBrightness:           80.0,
				MaxBrightness:           220.0,
			}
			// If request has resolution/skew thresholds, overlay them (optional fields)
			if ct := request.CustomThresholds; ct != nil {
				if ct.MaxSkewAngle != nil {
					t.MaxSkewAngle = *ct.MaxSkewAngle
				}
				// Use MinResolution if provided (this replaces MinWidth, MinHeight, MinTotalPixels)
				if ct.MinResolution != nil {
					// Set a default value for min resolution
					minRes := *ct.MinResolution
					t.MinWidth = minRes
					t.MinHeight = minRes
					t.MinTotalPixels = minRes * minRes
				}
			}
			return t
		}(),
		QualityChecks:     s.generateQualityChecks(result),
		OverallAssessment: s.generateOverallAssessment(result),
		ProcessingDetails: func() models.ProcessingDetails {
			mode := strings.ToLower(strings.TrimSpace(request.AnalysisMode))
			if mode == "" {
				if options.OCRMode {
					mode = "ocr"
				} else {
					mode = "quality"
				}
			}
			features := []string{"sharpness", "exposure", "color", "resolution"}
			skipped := []string{}
			if options.SkipQRDetection {
				skipped = append(skipped, "qr")
			} else {
				features = append(features, "qr")
			}
			if options.SkipWhiteBalance {
				skipped = append(skipped, "white_balance")
			}
			if options.SkipContourDetection {
				skipped = append(skipped, "contour_detection")
			}
			if options.SkipEdgeDetection {
				skipped = append(skipped, "edge_detection")
			}
			return models.ProcessingDetails{
				AnalysisMode:      mode,
				FeaturesAnalyzed:  features,
				SkippedFeatures:   skipped,
				ProcessingOptions: map[string]interface{}{"use_worker_pool": options.UseWorkerPool, "max_workers": options.MaxWorkers},
				PerformanceMetrics: models.PerformanceMetrics{},
			}
		}(),
		Errors: result.Errors,
	}

	// Add OCR analysis if available
	if result.OCRResult != nil {
		response.OCRAnalysis = &models.DetailedOCRAnalysis{
			OCRReadinessScore:  s.computeOCRReadiness(result, width, height),
			TextDetectionScore: s.computeTextDetectionScore(result.OCRResult),
			DocumentType:       "text",
			TextDensity:        s.computeTextDensity(result.OCRResult, width, height),
			EstimatedTextLines: s.estimateTextLines(result.OCRResult),
		}
	}

	return response
}

// Helper methods
func (s *imageAnalysisService) getImageDimensions(img image.Image) (int, int) {
	if img == nil {
		return 0, 0
	}
	bounds := img.Bounds()
	return bounds.Dx(), bounds.Dy()
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

// OCR Analysis Helper Functions

// computeOCRReadiness calculates OCR readiness score based on quality metrics
func (s *imageAnalysisService) computeOCRReadiness(result *models.AnalysisResult, width, height int) float64 {
	score := 100.0

	// Penalize for blur (most critical for OCR)
	if result.Quality.Blurry {
		score -= 40.0
	} else if result.Metrics.LaplacianVar < 500.0 {
		// Partial penalty for low sharpness
		score -= (500.0 - result.Metrics.LaplacianVar) / 500.0 * 20.0
	}

	// Penalize for brightness issues
	if result.Quality.IsTooDark {
		score -= 25.0
	} else if result.Quality.IsTooBright {
		score -= 20.0
	}

	// Penalize for low resolution
	if result.Quality.IsLowResolution {
		score -= 30.0
	} else if width*height < 1200000 { // Less than 1.2MP
		score -= 15.0
	}

	// Penalize for skew
	if result.Quality.IsSkewed {
		score -= 15.0
	}

	// Penalize for exposure issues
	if result.Quality.Overexposed {
		score -= 20.0
	}

	// Penalize for color issues
	if result.Quality.Oversaturated {
		score -= 10.0
	}
	if result.Quality.IncorrectWB {
		score -= 10.0
	}

	// Ensure score is within bounds
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

// computeTextDetectionScore calculates text detection score based on OCR confidence
func (s *imageAnalysisService) computeTextDetectionScore(ocrResult *models.OCRResult) float64 {
	if ocrResult == nil {
		return 0.0
	}

	// Base score on OCR confidence
	score := ocrResult.Confidence

	// Boost score if text was actually extracted
	if len(ocrResult.ExtractedText) > 0 {
		score += 10.0
	}

	// Boost score based on text length (more text = better detection)
	textLength := len(ocrResult.ExtractedText)
	if textLength > 100 {
		score += 15.0
	} else if textLength > 50 {
		score += 10.0
	} else if textLength > 10 {
		score += 5.0
	}

	// Penalize for high error rates
	if ocrResult.WER > 0.5 {
		score -= 20.0
	} else if ocrResult.WER > 0.3 {
		score -= 10.0
	}

	if ocrResult.CER > 0.3 {
		score -= 15.0
	} else if ocrResult.CER > 0.2 {
		score -= 8.0
	}

	// Ensure score is within bounds
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

// computeTextDensity estimates text density based on extracted text and image dimensions
func (s *imageAnalysisService) computeTextDensity(ocrResult *models.OCRResult, width, height int) float64 {
	if ocrResult == nil || len(ocrResult.ExtractedText) == 0 {
		return 0.0
	}

	// Calculate approximate text coverage
	// Assume average character takes about 12x16 pixels
	charPixels := 12 * 16
	totalTextPixels := len(ocrResult.ExtractedText) * charPixels
	totalImagePixels := width * height
	if totalImagePixels <= 0 {
		return 0.0
	}

	density := float64(totalTextPixels) / float64(totalImagePixels)

	// Cap density at reasonable maximum (0.8 for very dense text documents)
	if density > 0.8 {
		density = 0.8
	}

	return density
}

// estimateTextLines estimates number of text lines based on OCR text content
func (s *imageAnalysisService) estimateTextLines(ocrResult *models.OCRResult) int {
	if ocrResult == nil || len(ocrResult.ExtractedText) == 0 {
		return 0
	}

	text := ocrResult.ExtractedText

	// Count explicit newlines
	lines := 1
	for _, char := range text {
		if char == '\n' {
			lines++
		}
	}

	// If no newlines found, estimate based on text length
	if lines == 1 && len(text) > 0 {
		// Assume average line has about 60-80 characters
		avgCharsPerLine := 70
		estimatedLines := len(text) / avgCharsPerLine
		if estimatedLines > 1 {
			lines = estimatedLines
		}
	}

	return lines
}
