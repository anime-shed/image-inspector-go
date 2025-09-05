package service

import (
	"testing"

	"github.com/anime-shed/image-inspector-go/pkg/models"
)

func TestComputeOCRReadiness(t *testing.T) {
	service := &imageAnalysisService{}

	tests := []struct {
		name     string
		result   *models.AnalysisResult
		width    int
		height   int
		expected float64
	}{
		{
			name: "Perfect quality image",
			result: &models.AnalysisResult{
				Quality: models.Quality{
					Blurry:          false,
					IsTooDark:       false,
					IsTooBright:     false,
					IsLowResolution: false,
					IsSkewed:        false,
					Overexposed:     false,
					Oversaturated:   false,
					IncorrectWB:     false,
				},
				Metrics: models.ImageMetrics{
					LaplacianVar: 800.0,
				},
			},
			width:    1920,
			height:   1080,
			expected: 100.0,
		},
		{
			name: "Blurry image",
			result: &models.AnalysisResult{
				Quality: models.Quality{
					Blurry:          true,
					IsTooDark:       false,
					IsTooBright:     false,
					IsLowResolution: false,
					IsSkewed:        false,
					Overexposed:     false,
					Oversaturated:   false,
					IncorrectWB:     false,
				},
				Metrics: models.ImageMetrics{
					LaplacianVar: 50.0,
				},
			},
			width:    1920,
			height:   1080,
			expected: 60.0, // 100 - 40 (blur penalty)
		},
		{
			name: "Low resolution image",
			result: &models.AnalysisResult{
				Quality: models.Quality{
					Blurry:          false,
					IsTooDark:       false,
					IsTooBright:     false,
					IsLowResolution: true,
					IsSkewed:        false,
					Overexposed:     false,
					Oversaturated:   false,
					IncorrectWB:     false,
				},
				Metrics: models.ImageMetrics{
					LaplacianVar: 600.0,
				},
			},
			width:    640,
			height:   480,
			expected: 70.0, // 100 - 30 (low resolution penalty)
		},
		{
			name: "Multiple issues",
			result: &models.AnalysisResult{
				Quality: models.Quality{
					Blurry:          true,
					IsTooDark:       true,
					IsTooBright:     false,
					IsLowResolution: true,
					IsSkewed:        true,
					Overexposed:     false,
					Oversaturated:   false,
					IncorrectWB:     false,
				},
				Metrics: models.ImageMetrics{
					LaplacianVar: 30.0,
				},
			},
			width:    400,
			height:   300,
			expected: 0.0, // 100 - 40 (blur) - 25 (dark) - 30 (low res) - 15 (skew) = -10, capped at 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.computeOCRReadiness(tt.result, tt.width, tt.height)
			if result != tt.expected {
				t.Errorf("computeOCRReadiness() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestComputeTextDetectionScore(t *testing.T) {
	service := &imageAnalysisService{}

	tests := []struct {
		name      string
		ocrResult *models.OCRResult
		expected  float64
	}{
		{
			name:      "Nil OCR result",
			ocrResult: nil,
			expected:  0.0,
		},
		{
			name: "High confidence with long text",
			ocrResult: &models.OCRResult{
				ExtractedText: "This is a long text with more than 100 characters to test the text detection scoring algorithm properly.",
				Confidence:    85.0,
				WER:           0.1,
				CER:           0.05,
			},
			expected: 100.0, // 85 + 10 (has text) + 15 (long text) = 110, capped at 100
		},
		{
			name: "Medium confidence with short text",
			ocrResult: &models.OCRResult{
				ExtractedText: "Short text",
				Confidence:    70.0,
				WER:           0.2,
				CER:           0.1,
			},
			expected: 80.0, // 70 + 10 (has text) + 0 (short text < 50 chars)
		},
		{
			name: "High error rates",
			ocrResult: &models.OCRResult{
				ExtractedText: "Text with errors",
				Confidence:    80.0,
				WER:           0.6,
				CER:           0.4,
			},
			expected: 60.0, // 80 + 10 (has text) + 0 (short text) - 20 (high WER) - 15 (high CER) + 5 (text > 10 chars)
		},
		{
			name: "No extracted text",
			ocrResult: &models.OCRResult{
				ExtractedText: "",
				Confidence:    60.0,
				WER:           0.0,
				CER:           0.0,
			},
			expected: 60.0, // Just confidence score
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.computeTextDetectionScore(tt.ocrResult)
			if result != tt.expected {
				t.Errorf("computeTextDetectionScore() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestComputeTextDensity(t *testing.T) {
	service := &imageAnalysisService{}

	tests := []struct {
		name      string
		ocrResult *models.OCRResult
		width     int
		height    int
		expected  float64
	}{
		{
			name:      "Nil OCR result",
			ocrResult: nil,
			width:     1920,
			height:    1080,
			expected:  0.0,
		},
		{
			name: "Empty text",
			ocrResult: &models.OCRResult{
				ExtractedText: "",
			},
			width:    1920,
			height:   1080,
			expected: 0.0,
		},
		{
			name: "Short text in large image",
			ocrResult: &models.OCRResult{
				ExtractedText: "Hello",
			},
			width:    1920,
			height:   1080,
			expected: 0.00046, // Approximately (5 * 192) / (1920 * 1080)
		},
		{
			name: "Long text in small image",
			ocrResult: &models.OCRResult{
				ExtractedText: "This is a very long text that would have high density in a small image. " +
					"It contains many characters and should result in a higher density score when " +
					"calculated against a smaller image dimension.",
			},
			width:    400,
			height:   300,
			expected: 0.31, // Approximately (186 chars * 192 pixels) / (400 * 300) = 0.2976
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.computeTextDensity(tt.ocrResult, tt.width, tt.height)
			// Use approximate comparison for floating point
			if abs(result-tt.expected) > 0.001 {
				t.Errorf("computeTextDensity() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEstimateTextLines(t *testing.T) {
	service := &imageAnalysisService{}

	tests := []struct {
		name      string
		ocrResult *models.OCRResult
		expected  int
	}{
		{
			name:      "Nil OCR result",
			ocrResult: nil,
			expected:  0,
		},
		{
			name: "Empty text",
			ocrResult: &models.OCRResult{
				ExtractedText: "",
			},
			expected: 0,
		},
		{
			name: "Single line text",
			ocrResult: &models.OCRResult{
				ExtractedText: "This is a single line of text",
			},
			expected: 1,
		},
		{
			name: "Multi-line text with newlines",
			ocrResult: &models.OCRResult{
				ExtractedText: "First line\nSecond line\nThird line",
			},
			expected: 3,
		},
		{
			name: "Long text without newlines",
			ocrResult: &models.OCRResult{
				ExtractedText: "This is a very long text that would span multiple lines if it were displayed in a typical document format. " +
					"It contains more than 70 characters per estimated line, so it should be estimated as multiple lines based on character count.",
			},
			expected: 3, // Approximately 210 characters / 70 = 3 lines
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.estimateTextLines(tt.ocrResult)
			if result != tt.expected {
				t.Errorf("estimateTextLines() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Helper function for floating point comparison
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
