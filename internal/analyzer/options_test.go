package analyzer

import (
	"testing"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	// Verify default values
	if opts.OCRMode {
		t.Error("Expected OCRMode to be false by default")
	}
	if opts.FastMode {
		t.Error("Expected FastMode to be false by default")
	}
	if !opts.QualityMode {
		t.Error("Expected QualityMode to be true by default")
	}
	if opts.BlurThreshold != 100.0 {
		t.Errorf("Expected BlurThreshold to be 100.0, got %f", opts.BlurThreshold)
	}
	if opts.OverexposureThreshold != 0.95 {
		t.Errorf("Expected OverexposureThreshold to be 0.95, got %f", opts.OverexposureThreshold)
	}
	if opts.OversaturationThreshold != 0.9 {
		t.Errorf("Expected OversaturationThreshold to be 0.9, got %f", opts.OversaturationThreshold)
	}
}

func TestOCROptions(t *testing.T) {
	opts := OCROptions()

	// Verify OCR-specific values
	if !opts.OCRMode {
		t.Error("Expected OCRMode to be true for OCR options")
	}
	if !opts.QualityMode {
		t.Error("Expected QualityMode to be true for OCR options")
	}
	if opts.BlurThreshold != 300.0 {
		t.Errorf("Expected BlurThreshold to be 300.0 for OCR, got %f", opts.BlurThreshold)
	}
	if !opts.SkipQRDetection {
		t.Error("Expected SkipQRDetection to be true for OCR options")
	}
	if opts.OCRLanguage != "eng" {
		t.Errorf("Expected OCRLanguage to be 'eng', got %s", opts.OCRLanguage)
	}
}

func TestFastOptions(t *testing.T) {
	opts := FastOptions()

	// Verify fast mode values
	if !opts.FastMode {
		t.Error("Expected FastMode to be true for fast options")
	}
	if opts.QualityMode {
		t.Error("Expected QualityMode to be false for fast options")
	}
	if !opts.SkipContourDetection {
		t.Error("Expected SkipContourDetection to be true for fast options")
	}
	if !opts.SkipEdgeDetection {
		t.Error("Expected SkipEdgeDetection to be true for fast options")
	}
	if !opts.SkipWhiteBalance {
		t.Error("Expected SkipWhiteBalance to be true for fast options")
	}
}

func TestQualityOptions(t *testing.T) {
	opts := QualityOptions()

	// Verify quality mode values
	if !opts.QualityMode {
		t.Error("Expected QualityMode to be true for quality options")
	}
	if opts.BlurThreshold != 400.0 {
		t.Errorf("Expected BlurThreshold to be 400.0 for quality mode, got %f", opts.BlurThreshold)
	}
	if opts.OverexposureThreshold != 0.9 {
		t.Errorf("Expected OverexposureThreshold to be 0.9 for quality mode, got %f", opts.OverexposureThreshold)
	}
	if opts.OversaturationThreshold != 0.85 {
		t.Errorf("Expected OversaturationThreshold to be 0.85 for quality mode, got %f", opts.OversaturationThreshold)
	}
}

func TestWithOCR(t *testing.T) {
	opts := DefaultOptions().WithOCR("test text")

	if !opts.OCRMode {
		t.Error("Expected OCRMode to be true after WithOCR")
	}
	if opts.OCRExpectedText != "test text" {
		t.Errorf("Expected OCRExpectedText to be 'test text', got %s", opts.OCRExpectedText)
	}
	if !opts.QualityMode {
		t.Error("Expected QualityMode to be true after WithOCR")
	}
}

func TestWithCustomThresholds(t *testing.T) {
	opts := DefaultOptions().WithCustomThresholds(250.0, 0.8, 0.7)

	if opts.BlurThreshold != 250.0 {
		t.Errorf("Expected BlurThreshold to be 250.0, got %f", opts.BlurThreshold)
	}
	if opts.OverexposureThreshold != 0.8 {
		t.Errorf("Expected OverexposureThreshold to be 0.8, got %f", opts.OverexposureThreshold)
	}
	if opts.OversaturationThreshold != 0.7 {
		t.Errorf("Expected OversaturationThreshold to be 0.7, got %f", opts.OversaturationThreshold)
	}
}

func TestWithFastMode(t *testing.T) {
	opts := DefaultOptions().WithFastMode()

	if !opts.FastMode {
		t.Error("Expected FastMode to be true after WithFastMode")
	}
	if opts.QualityMode {
		t.Error("Expected QualityMode to be false after WithFastMode")
	}
	if !opts.SkipContourDetection {
		t.Error("Expected SkipContourDetection to be true after WithFastMode")
	}
	if !opts.SkipEdgeDetection {
		t.Error("Expected SkipEdgeDetection to be true after WithFastMode")
	}
}

func TestWithoutQRDetection(t *testing.T) {
	opts := DefaultOptions().WithoutQRDetection()

	if !opts.SkipQRDetection {
		t.Error("Expected SkipQRDetection to be true after WithoutQRDetection")
	}
}

func TestChainedOptions(t *testing.T) {
	// Test chaining multiple option methods
	opts := DefaultOptions().
		WithOCR("sample text").
		WithCustomThresholds(200.0, 0.85, 0.75).
		WithoutQRDetection()

	if !opts.OCRMode {
		t.Error("Expected OCRMode to be true")
	}
	if opts.OCRExpectedText != "sample text" {
		t.Errorf("Expected OCRExpectedText to be 'sample text', got %s", opts.OCRExpectedText)
	}
	if opts.BlurThreshold != 200.0 {
		t.Errorf("Expected BlurThreshold to be 200.0, got %f", opts.BlurThreshold)
	}
	if opts.OverexposureThreshold != 0.85 {
		t.Errorf("Expected OverexposureThreshold to be 0.85, got %f", opts.OverexposureThreshold)
	}
	if opts.OversaturationThreshold != 0.75 {
		t.Errorf("Expected OversaturationThreshold to be 0.75, got %f", opts.OversaturationThreshold)
	}
	if !opts.SkipQRDetection {
		t.Error("Expected SkipQRDetection to be true")
	}
}