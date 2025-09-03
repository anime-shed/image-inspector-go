package analyzer

import (
	"image"
	"image/color"
	"testing"
	"time"
)

// createTestImage creates a simple test image for testing purposes
func createTestImage(width, height int, fillColor color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, fillColor)
		}
	}
	return img
}

// createGradientImage creates a gradient test image
func createGradientImage(width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Create a gradient from black to white
			intensity := uint8((x + y) * 255 / (width + height))
			img.Set(x, y, color.RGBA{intensity, intensity, intensity, 255})
		}
	}
	return img
}

func TestNewImageAnalyzer(t *testing.T) {
	analyzer, err := NewImageAnalyzer()
	if err != nil {
		t.Fatalf("Failed to create image analyzer: %v", err)
	}
	if analyzer == nil {
		t.Fatal("Expected non-nil analyzer")
	}
}

func TestAnalyze_BasicImage(t *testing.T) {
	analyzer, err := NewImageAnalyzer()
	if err != nil {
		t.Fatalf("Failed to create image analyzer: %v", err)
	}

	// Create a test image
	img := createTestImage(800, 600, color.RGBA{128, 128, 128, 255})

	// Analyze the image
	result := analyzer.Analyze(img, false)

	// Verify basic fields are set
	if result.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}
	if result.ProcessingTimeSec <= 0 {
		t.Error("Expected processing time to be positive")
	}
	if result.Metrics.LaplacianVar < 0 {
		t.Error("Expected non-negative Laplacian variance")
	}
	if result.Metrics.AvgLuminance < 0 || result.Metrics.AvgLuminance > 1 {
		t.Errorf("Expected luminance between 0 and 1, got %f", result.Metrics.AvgLuminance)
	}
	if result.Metrics.AvgSaturation < 0 || result.Metrics.AvgSaturation > 1 {
		t.Errorf("Expected saturation between 0 and 1, got %f", result.Metrics.AvgSaturation)
	}
}

func TestAnalyze_OCRMode(t *testing.T) {
	analyzer, err := NewImageAnalyzer()
	if err != nil {
		t.Fatalf("Failed to create image analyzer: %v", err)
	}

	// Create a test image
	img := createTestImage(1200, 800, color.RGBA{200, 200, 200, 255})

	// Analyze with OCR mode
	result := analyzer.Analyze(img, true)

	// Verify OCR-specific fields are set
	if result.Metrics.Resolution == "" {
		t.Error("Expected resolution to be set in OCR mode")
	}
	if result.Metrics.Brightness <= 0 {
		t.Error("Expected brightness to be calculated in OCR mode")
	}
	if result.Metrics.NumContours < 0 {
		t.Error("Expected non-negative contour count")
	}
}

func TestAnalyze_LowResolutionImage(t *testing.T) {
	analyzer, err := NewImageAnalyzer()
	if err != nil {
		t.Fatalf("Failed to create image analyzer: %v", err)
	}

	// Create a low resolution image
	img := createTestImage(400, 300, color.RGBA{128, 128, 128, 255})

	// Analyze with OCR mode to trigger resolution check
	result := analyzer.Analyze(img, true)

	// Should detect low resolution
	if !result.Quality.IsLowResolution {
		t.Error("Expected low resolution to be detected")
	}
}

func TestAnalyze_BrightImage(t *testing.T) {
	analyzer, err := NewImageAnalyzer()
	if err != nil {
		t.Fatalf("Failed to create image analyzer: %v", err)
	}

	// Create a very bright image
	img := createTestImage(800, 600, color.RGBA{250, 250, 250, 255})

	// Analyze with OCR mode
	result := analyzer.Analyze(img, true)

	// Should detect brightness issues
	if !result.Quality.IsTooBright {
		t.Error("Expected bright image to be detected")
	}
	if result.Metrics.Brightness < 200 {
		t.Errorf("Expected high brightness value, got %f", result.Metrics.Brightness)
	}
}

func TestAnalyze_DarkImage(t *testing.T) {
	analyzer, err := NewImageAnalyzer()
	if err != nil {
		t.Fatalf("Failed to create image analyzer: %v", err)
	}

	// Create a very dark image
	img := createTestImage(800, 600, color.RGBA{20, 20, 20, 255})

	// Analyze with OCR mode
	result := analyzer.Analyze(img, true)

	// Should detect darkness issues
	if !result.Quality.IsTooDark {
		t.Error("Expected dark image to be detected")
	}
	if result.Metrics.Brightness > 80 {
		t.Errorf("Expected low brightness value, got %f", result.Metrics.Brightness)
	}
}

func TestAnalyzeWithOCR(t *testing.T) {
	analyzer, err := NewImageAnalyzer()
	if err != nil {
		t.Fatalf("Failed to create image analyzer: %v", err)
	}

	// Create a test image
	img := createTestImage(800, 600, color.RGBA{128, 128, 128, 255})
	expectedText := "Hello World"

	// Analyze with OCR
	result := analyzer.AnalyzeWithOCR(img, expectedText)

	// Verify OCR-specific fields
	if result.OCRResult == nil || result.OCRResult.ExpectedText != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, result.OCRResult.ExpectedText)
	}
	// OCR is not implemented yet, so we expect an error message
	if result.OCRResult == nil || result.OCRResult.OCRError == "" {
		t.Error("Expected OCR error message since OCR is not implemented")
	}
}

func TestAnalyze_Performance(t *testing.T) {
	analyzer, err := NewImageAnalyzer()
	if err != nil {
		t.Fatalf("Failed to create image analyzer: %v", err)
	}

	// Create a reasonably sized test image
	img := createGradientImage(1000, 800)

	// Measure analysis time
	start := time.Now()
	result := analyzer.Analyze(img, true)
	duration := time.Since(start)

	// Analysis should complete within reasonable time (5 seconds)
	if duration > 5*time.Second {
		t.Errorf("Analysis took too long: %v", duration)
	}

	// Verify processing time is recorded
	if result.ProcessingTimeSec <= 0 {
		t.Error("Expected positive processing time")
	}
	if result.ProcessingTimeSec > 5.0 {
		t.Errorf("Processing time seems too high: %f seconds", result.ProcessingTimeSec)
	}
}

func TestAnalyze_MultipleImages(t *testing.T) {
	analyzer, err := NewImageAnalyzer()
	if err != nil {
		t.Fatalf("Failed to create image analyzer: %v", err)
	}

	// Test with multiple different images
	testCases := []struct {
		name  string
		color color.RGBA
		width int
		height int
	}{
		{"Gray Image", color.RGBA{128, 128, 128, 255}, 800, 600},
		{"Red Image", color.RGBA{255, 0, 0, 255}, 800, 600},
		{"Blue Image", color.RGBA{0, 0, 255, 255}, 800, 600},
		{"Small Image", color.RGBA{128, 128, 128, 255}, 400, 300},
		{"Large Image", color.RGBA{128, 128, 128, 255}, 1600, 1200},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			img := createTestImage(tc.width, tc.height, tc.color)
			result := analyzer.Analyze(img, false)

			// Basic validation
			if result.Timestamp.IsZero() {
				t.Error("Expected timestamp to be set")
			}
			if result.ProcessingTimeSec <= 0 {
				t.Error("Expected positive processing time")
			}
		})
	}
}