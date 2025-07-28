package service

import (
	"context"
	"go-image-inspector/internal/analyzer"
	"go-image-inspector/internal/repository"
	apperrors "go-image-inspector/internal/errors"
)

// ImageAnalysisService defines the interface for image analysis business logic
type ImageAnalysisService interface {
	AnalyzeImage(ctx context.Context, imageURL string, isOCR bool) (*ImageAnalysisResponse, error)
	AnalyzeImageWithOCR(ctx context.Context, imageURL string, expectedText string) (*ImageAnalysisResponse, error)
	ValidateImageURL(imageURL string) error
}

// ImageAnalysisResponse represents the response from image analysis
type ImageAnalysisResponse struct {
	ImageURL          string     `json:"image_url"`
	Timestamp         string     `json:"timestamp"`
	ProcessingTimeSec float64    `json:"processing_time_sec"`
	Quality           Quality    `json:"quality"`
	Metrics           Metrics    `json:"metrics"`
	OCRResult         *OCRResult `json:"ocr_result,omitempty"`
	Errors            []string   `json:"errors,omitempty"`
}

// Quality represents image quality assessment
type Quality struct {
	Overexposed   bool `json:"overexposed"`
	Oversaturated bool `json:"oversaturated"`
	IncorrectWB   bool `json:"incorrect_wb"`
	Blurry        bool `json:"blurry"`
	IsValid       bool `json:"is_valid"`
}

// Metrics represents image analysis metrics
type Metrics struct {
	LaplacianVar      float64    `json:"laplacian_var"`
	AvgLuminance      float64    `json:"avg_luminance"`
	AvgSaturation     float64    `json:"avg_saturation"`
	ChannelBalance    [3]float64 `json:"channel_balance"`
}

// OCRResult represents OCR analysis results
type OCRResult struct {
	ExtractedText string  `json:"extracted_text"`
	ExpectedText  string  `json:"expected_text,omitempty"`
	Confidence    float64 `json:"confidence"`
	MatchScore    float64 `json:"match_score,omitempty"`
}

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

// AnalyzeImage performs image analysis
func (s *imageAnalysisService) AnalyzeImage(ctx context.Context, imageURL string, isOCR bool) (*ImageAnalysisResponse, error) {
	// Validate URL
	if err := s.ValidateImageURL(imageURL); err != nil {
		return nil, apperrors.NewValidationError("invalid image URL", err)
	}

	// Fetch image
	img, err := s.imageRepo.FetchImage(ctx, imageURL)
	if err != nil {
		return nil, apperrors.NewNetworkError("failed to fetch image", err)
	}

	// Analyze image
	result := s.analyzer.Analyze(img, isOCR)

	// Convert to service response
	response := s.convertToResponse(imageURL, result)
	
	return response, nil
}

// AnalyzeImageWithOCR performs OCR-specific image analysis
func (s *imageAnalysisService) AnalyzeImageWithOCR(ctx context.Context, imageURL string, expectedText string) (*ImageAnalysisResponse, error) {
	// Validate URL
	if err := s.ValidateImageURL(imageURL); err != nil {
		return nil, apperrors.NewValidationError("invalid image URL", err)
	}

	// Fetch image
	img, err := s.imageRepo.FetchImage(ctx, imageURL)
	if err != nil {
		return nil, apperrors.NewNetworkError("failed to fetch image", err)
	}

	// Analyze image with OCR
	result := s.analyzer.AnalyzeWithOCR(img, expectedText)

	// Convert to service response
	response := s.convertToResponse(imageURL, result)
	
	// Add OCR-specific data
	if result.OCRText != "" || result.ExpectedText != "" {
		response.OCRResult = &OCRResult{
			ExtractedText: result.OCRText,
			ExpectedText:  result.ExpectedText,
			Confidence:    0.0, // TODO: Implement confidence calculation
			MatchScore:    0.0, // TODO: Implement match score calculation
		}
	}
	
	return response, nil
}

// ValidateImageURL validates the image URL
func (s *imageAnalysisService) ValidateImageURL(imageURL string) error {
	return s.imageRepo.ValidateImageURL(imageURL)
}

// convertToResponse converts analyzer result to service response
func (s *imageAnalysisService) convertToResponse(imageURL string, result analyzer.AnalysisResult) *ImageAnalysisResponse {
	return &ImageAnalysisResponse{
		ImageURL:          imageURL,
		Timestamp:         result.Timestamp,
		ProcessingTimeSec: result.ProcessingTimeSec,
		Quality: Quality{
			Overexposed:   result.Overexposed,
			Oversaturated: result.Oversaturated,
			IncorrectWB:   result.IncorrectWB,
			Blurry:        result.Blurry,
			IsValid:       len(result.Errors) == 0,
		},
		Metrics: Metrics{
			LaplacianVar:   result.LaplacianVar,
			AvgLuminance:   result.AvgLuminance,
			AvgSaturation:  result.AvgSaturation,
			ChannelBalance: result.ChannelBalance,
		},
		Errors: result.Errors,
	}
}