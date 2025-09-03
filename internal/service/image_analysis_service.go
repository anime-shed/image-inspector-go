package service

import (
	"context"
	"go-image-inspector/internal/analyzer"
	"go-image-inspector/internal/repository"
	apperrors "go-image-inspector/internal/errors"
	"go-image-inspector/pkg/models"
)

// ImageAnalysisService defines the interface for image analysis operations
type ImageAnalysisService interface {
	// Legacy methods for backward compatibility
	AnalyzeImage(ctx context.Context, imageURL string, isOCR bool) (*models.ImageAnalysisResponse, error)
	AnalyzeImageWithOCR(ctx context.Context, imageURL string, expectedText string) (*models.ImageAnalysisResponse, error)
	
	// New options-based method
	AnalyzeImageWithOptions(ctx context.Context, imageURL string, options analyzer.AnalysisOptions) (*models.ImageAnalysisResponse, error)
	
	ValidateImageURL(imageURL string) error
}

// ImageAnalysisResponse is now an alias to the shared models.ImageAnalysisResponse
type ImageAnalysisResponse = models.ImageAnalysisResponse

// Quality is now an alias to the shared models.Quality
type Quality = models.Quality

// Metrics is now an alias to the shared models.ImageMetrics
type Metrics = models.ImageMetrics

// OCRResult is now an alias to the shared models.OCRResult
type OCRResult = models.OCRResult

// imageAnalysisService implements ImageAnalysisService
type imageAnalysisService struct {
	imageRepo repository.ImageRepository
	analyzer  analyzer.ImageAnalyzer
}

// NewImageAnalysisService creates a new image analysis service
func NewImageAnalysisService(imageRepo repository.ImageRepository, analyzer analyzer.ImageAnalyzer) ImageAnalysisService {
	return &imageAnalysisService{
		imageRepo: imageRepo,
		analyzer:  analyzer,
	}
}

// AnalyzeImage performs image analysis (legacy method for backward compatibility)
func (s *imageAnalysisService) AnalyzeImage(ctx context.Context, imageURL string, isOCR bool) (*ImageAnalysisResponse, error) {
	options := analyzer.DefaultOptions()
	options.OCRMode = isOCR
	return s.AnalyzeImageWithOptions(ctx, imageURL, options)
}

// AnalyzeImageWithOptions performs image analysis with flexible configuration
func (s *imageAnalysisService) AnalyzeImageWithOptions(ctx context.Context, imageURL string, options analyzer.AnalysisOptions) (*ImageAnalysisResponse, error) {
	// Validate URL
	if err := s.ValidateImageURL(imageURL); err != nil {
		return nil, apperrors.NewValidationError("invalid image URL", err)
	}

	// Fetch image
	img, err := s.imageRepo.FetchImage(ctx, imageURL)
	if err != nil {
		return nil, apperrors.NewNetworkError("failed to fetch image", err)
	}

	// Analyze image with options
	result := s.analyzer.AnalyzeWithOptions(img, options)

	// Convert to service response
	response := s.convertToResponse(imageURL, &result)
	
	return response, nil
}

// AnalyzeImageWithOCR performs OCR-specific image analysis (legacy method for backward compatibility)
func (s *imageAnalysisService) AnalyzeImageWithOCR(ctx context.Context, imageURL string, expectedText string) (*ImageAnalysisResponse, error) {
	options := analyzer.OCROptions()
	options.OCRExpectedText = expectedText
	return s.AnalyzeImageWithOptions(ctx, imageURL, options)
}

// ValidateImageURL validates the image URL
func (s *imageAnalysisService) ValidateImageURL(imageURL string) error {
	return s.imageRepo.ValidateImageURL(imageURL)
}

// convertToResponse converts analyzer result to service response
func (s *imageAnalysisService) convertToResponse(imageURL string, result *models.AnalysisResult) *ImageAnalysisResponse {
	response := &ImageAnalysisResponse{
		ImageURL:          imageURL,
		Timestamp:         result.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
		ProcessingTimeSec: result.ProcessingTimeSec,
		Quality: Quality{
			Overexposed:   result.Quality.Overexposed,
			Oversaturated: result.Quality.Oversaturated,
			IncorrectWB:   result.Quality.IncorrectWB,
			Blurry:        result.Quality.Blurry,
			IsValid:       len(result.Errors) == 0,
		},
		Metrics: Metrics{
			LaplacianVar:   result.Metrics.LaplacianVar,
			AvgLuminance:   result.Metrics.AvgLuminance,
			AvgSaturation:  result.Metrics.AvgSaturation,
			ChannelBalance: result.Metrics.ChannelBalance,
		},
		Errors: result.Errors,
	}
	
	// Add OCR result if available
	if result.OCRResult != nil {
		response.OCRResult = &OCRResult{
			ExtractedText: result.OCRResult.ExtractedText,
			ExpectedText:  result.OCRResult.ExpectedText,
			Confidence:    result.OCRResult.Confidence,
			MatchScore:    result.OCRResult.MatchScore,
		}
	}
	
	return response
}