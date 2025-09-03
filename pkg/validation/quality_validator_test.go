package validation

import (
	"testing"
)

func TestNewQualityValidator(t *testing.T) {
	validator := NewQualityValidator()
	if validator == nil {
		t.Fatal("Expected non-nil quality validator")
	}

	// Check default thresholds are set
	expected := DefaultQualityThresholds().MinLaplacianVariance
	if validator.thresholds.MinLaplacianVariance != expected {
		t.Errorf("Expected MinLaplacianVariance to be %f, got %f", expected, validator.thresholds.MinLaplacianVariance)
	}
}

func TestNewQualityValidatorWithThresholds(t *testing.T) {
	customThresholds := QualityThresholds{
		MinLaplacianVariance: 500.0,
		MinBrightness:        100.0,
		MaxBrightness:        200.0,
	}

	validator := NewQualityValidatorWithThresholds(customThresholds)
	if validator.thresholds.MinLaplacianVariance != 500.0 {
		t.Errorf("Expected custom MinLaplacianVariance to be 500.0, got %f", validator.thresholds.MinLaplacianVariance)
	}
}

func TestValidateBasicQuality_HighQuality(t *testing.T) {
	validator := NewQualityValidator()

	// Create metrics for a high-quality image
	metrics := ImageQualityMetrics{
		Width:          1920,
		Height:         1080,
		LaplacianVar:   1000.0,                       // Good sharpness
		Brightness:     150.0,                        // Good brightness
		AvgLuminance:   0.5,                          // Good luminance
		AvgSaturation:  0.3,                          // Good saturation
		ChannelBalance: [3]float64{0.33, 0.33, 0.34}, // Good balance
		Overexposed:    false,
		Oversaturated:  false,
		IncorrectWB:    false,
		IsTooDark:      false,
		IsTooBright:    false,
	}

	issues := validator.ValidateBasicQuality(metrics)

	// Should have no quality issues for a good image
	if len(issues) > 0 {
		t.Errorf("Expected no quality issues for high-quality image, got: %v", issues)
	}
}

func TestValidateBasicQuality_Blurry(t *testing.T) {
	validator := NewQualityValidator()

	metrics := ImageQualityMetrics{
		LaplacianVar:   50.0, // Below threshold (100)
		AvgLuminance:   0.5,
		AvgSaturation:  0.3,
		ChannelBalance: [3]float64{0.33, 0.33, 0.34},
	}

	issues := validator.ValidateBasicQuality(metrics)

	// Should detect blurriness
	hasBlurrinessIssue := false
	for _, issue := range issues {
		if issue.Type == "blurriness" {
			hasBlurrinessIssue = true
			if issue.Severity != "error" {
				t.Errorf("Expected blurriness to be error severity, got %s", issue.Severity)
			}
			if issue.ActualValue != 50.0 {
				t.Errorf("Expected actual value to be 50.0, got %f", issue.ActualValue)
			}
			break
		}
	}

	if !hasBlurrinessIssue {
		t.Error("Expected blurriness issue for blurry image")
	}
}

func TestValidateBasicQuality_Overexposed(t *testing.T) {
	validator := NewQualityValidator()

	metrics := ImageQualityMetrics{
		LaplacianVar:   1000.0,
		Overexposed:    true, // Overexposed
		AvgLuminance:   0.5,
		AvgSaturation:  0.3,
		ChannelBalance: [3]float64{0.33, 0.33, 0.34},
	}

	issues := validator.ValidateBasicQuality(metrics)

	// Should detect overexposure
	hasOverexposureIssue := false
	for _, issue := range issues {
		if issue.Type == "overexposure" {
			hasOverexposureIssue = true
			if issue.Severity != "error" {
				t.Errorf("Expected overexposure to be error severity, got %s", issue.Severity)
			}
			break
		}
	}

	if !hasOverexposureIssue {
		t.Error("Expected overexposure issue for overexposed image")
	}
}

func TestValidateBasicQuality_LowLuminance(t *testing.T) {
	validator := NewQualityValidator()

	metrics := ImageQualityMetrics{
		LaplacianVar:   1000.0,
		AvgLuminance:   0.1, // Below threshold (0.2)
		AvgSaturation:  0.3,
		ChannelBalance: [3]float64{0.33, 0.33, 0.34},
	}

	issues := validator.ValidateBasicQuality(metrics)

	// Should detect low luminance
	hasLowLuminanceIssue := false
	for _, issue := range issues {
		if issue.Type == "low_luminance" {
			hasLowLuminanceIssue = true
			if issue.ActualValue != 0.1 {
				t.Errorf("Expected actual value to be 0.1, got %f", issue.ActualValue)
			}
			break
		}
	}

	if !hasLowLuminanceIssue {
		t.Error("Expected low luminance issue for dark image")
	}
}

func TestValidateBasicQuality_ChannelImbalance(t *testing.T) {
	validator := NewQualityValidator()

	metrics := ImageQualityMetrics{
		LaplacianVar:   1000.0,
		AvgLuminance:   0.5,
		AvgSaturation:  0.3,
		ChannelBalance: [3]float64{0.8, 0.1, 0.1}, // High imbalance
	}

	issues := validator.ValidateBasicQuality(metrics)

	// Should detect channel imbalance
	hasChannelImbalanceIssue := false
	for _, issue := range issues {
		if issue.Type == "channel_imbalance" {
			hasChannelImbalanceIssue = true
			if issue.Severity != "warning" {
				t.Errorf("Expected channel imbalance to be warning severity, got %s", issue.Severity)
			}
			break
		}
	}

	if !hasChannelImbalanceIssue {
		t.Error("Expected channel imbalance issue for imbalanced image")
	}
}

func TestValidateOCRQuality_HighQuality(t *testing.T) {
	validator := NewQualityValidator()

	skewAngle := 2.0
	metrics := ImageQualityMetrics{
		Width:            1920,
		Height:           1080,
		LaplacianVar:     1000.0,                       // Good sharpness for OCR
		Brightness:       150.0,                        // Good brightness
		AvgLuminance:     0.5,                          // Good luminance
		AvgSaturation:    0.3,                          // Good saturation
		ChannelBalance:   [3]float64{0.33, 0.33, 0.34}, // Good balance
		Overexposed:      false,
		Oversaturated:    false,
		IncorrectWB:      false,
		IsTooDark:        false,
		IsTooBright:      false,
		IsSkewed:         false,
		HasDocumentEdges: true,
		SkewAngle:        &skewAngle, // Minimal skew
	}

	issues := validator.ValidateOCRQuality(metrics)

	// Should have no quality issues for a good OCR image
	if len(issues) > 0 {
		t.Errorf("Expected no quality issues for high-quality OCR image, got: %v", issues)
	}
}

func TestValidateOCRQuality_LowResolution(t *testing.T) {
	validator := NewQualityValidator()

	metrics := ImageQualityMetrics{
		Width:            400, // Below threshold (800)
		Height:           600, // Below threshold (1000)
		LaplacianVar:     1000.0,
		AvgLuminance:     0.5,
		AvgSaturation:    0.3,
		ChannelBalance:   [3]float64{0.33, 0.33, 0.34},
		HasDocumentEdges: true,
	}

	issues := validator.ValidateOCRQuality(metrics)

	// Should detect low resolution
	hasLowResolutionIssue := false
	for _, issue := range issues {
		if issue.Type == "low_resolution" {
			hasLowResolutionIssue = true
			if issue.Severity != "error" {
				t.Errorf("Expected low resolution to be error severity, got %s", issue.Severity)
			}
			break
		}
	}

	if !hasLowResolutionIssue {
		t.Error("Expected low resolution issue for small image")
	}
}

func TestValidateOCRQuality_Skewed(t *testing.T) {
	validator := NewQualityValidator()

	skewAngle := 10.0 // Above threshold (5.0)
	metrics := ImageQualityMetrics{
		Width:            1920,
		Height:           1080,
		LaplacianVar:     1000.0,
		AvgLuminance:     0.5,
		AvgSaturation:    0.3,
		ChannelBalance:   [3]float64{0.33, 0.33, 0.34},
		HasDocumentEdges: true,
		SkewAngle:        &skewAngle,
	}

	issues := validator.ValidateOCRQuality(metrics)

	// Should detect skew
	hasSkewIssue := false
	for _, issue := range issues {
		if issue.Type == "skew" {
			hasSkewIssue = true
			if issue.Severity != "warning" {
				t.Errorf("Expected skew to be warning severity, got %s", issue.Severity)
			}
			if issue.ActualValue != 10.0 {
				t.Errorf("Expected actual value to be 10.0, got %f", issue.ActualValue)
			}
			break
		}
	}

	if !hasSkewIssue {
		t.Error("Expected skew issue for skewed image")
	}
}

func TestValidateOCRQuality_NoDocumentEdges(t *testing.T) {
	validator := NewQualityValidator()

	metrics := ImageQualityMetrics{
		Width:            1920,
		Height:           1080,
		LaplacianVar:     1000.0,
		AvgLuminance:     0.5,
		AvgSaturation:    0.3,
		ChannelBalance:   [3]float64{0.33, 0.33, 0.34},
		HasDocumentEdges: false, // No document edges
	}

	issues := validator.ValidateOCRQuality(metrics)

	// Should detect missing document edges
	hasDocumentEdgesIssue := false
	for _, issue := range issues {
		if issue.Type == "document_edges" {
			hasDocumentEdgesIssue = true
			if issue.Severity != "error" {
				t.Errorf("Expected document edges to be error severity, got %s", issue.Severity)
			}
			break
		}
	}

	if !hasDocumentEdgesIssue {
		t.Error("Expected document edges issue for image without document edges")
	}
}

func TestConvertIssuesToMessages(t *testing.T) {
	validator := NewQualityValidator()

	issues := []QualityIssue{
		{Type: "blurriness", Message: "Image is blurry", Severity: "error"},
		{Type: "overexposure", Message: "Image is overexposed", Severity: "error"},
		{Type: "skew", Message: "Image is skewed", Severity: "warning"},
	}

	messages := validator.ConvertIssuesToMessages(issues)

	expectedMessages := []string{
		"Image is blurry",
		"Image is overexposed",
		"Image is skewed",
	}

	if len(messages) != len(expectedMessages) {
		t.Errorf("Expected %d messages, got %d", len(expectedMessages), len(messages))
	}

	for i, expected := range expectedMessages {
		if messages[i] != expected {
			t.Errorf("Expected message '%s', got '%s'", expected, messages[i])
		}
	}
}

func TestHasCriticalIssues(t *testing.T) {
	validator := NewQualityValidator()

	// Test with critical issues
	criticalIssues := []QualityIssue{
		{Type: "blurriness", Message: "Image is blurry", Severity: "error"},
		{Type: "skew", Message: "Image is skewed", Severity: "warning"},
	}

	if !validator.HasCriticalIssues(criticalIssues) {
		t.Error("Expected to have critical issues when error severity present")
	}

	// Test without critical issues
	nonCriticalIssues := []QualityIssue{
		{Type: "skew", Message: "Image is skewed", Severity: "warning"},
		{Type: "saturation", Message: "Low saturation", Severity: "info"},
	}

	if validator.HasCriticalIssues(nonCriticalIssues) {
		t.Error("Expected no critical issues when only warnings and info present")
	}
}

func TestDefaultQualityThresholds(t *testing.T) {
	thresholds := DefaultQualityThresholds()

	// Verify some key thresholds
	if thresholds.MinLaplacianVariance != 100.0 {
		t.Errorf("Expected MinLaplacianVariance to be 100.0, got %f", thresholds.MinLaplacianVariance)
	}
	if thresholds.MinLaplacianVarianceForOCR != 500.0 {
		t.Errorf("Expected MinLaplacianVarianceForOCR to be 500.0, got %f", thresholds.MinLaplacianVarianceForOCR)
	}
	if thresholds.MinWidth != 800 {
		t.Errorf("Expected MinWidth to be 800, got %d", thresholds.MinWidth)
	}
	if thresholds.MinHeight != 1000 {
		t.Errorf("Expected MinHeight to be 1000, got %d", thresholds.MinHeight)
	}
}
