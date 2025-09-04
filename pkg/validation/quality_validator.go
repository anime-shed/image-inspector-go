package validation

import (
	"math"
)

// QualityThresholds defines configurable thresholds for quality validation
type QualityThresholds struct {
	// Sharpness thresholds
	MinLaplacianVariance       float64
	MaxLaplacianVariance       float64
	MinLaplacianVarianceForOCR float64

	// Brightness thresholds
	MinBrightness float64
	MaxBrightness float64

	// Luminance thresholds
	MinLuminance float64
	MaxLuminance float64

	// Saturation thresholds
	MinSaturation float64

	// Channel balance threshold
	MaxChannelImbalance float64

	// Skew threshold (in degrees)
	MaxSkewAngle float64

	// Resolution thresholds
	MinWidth       int
	MinHeight      int
	MinTotalPixels int
}

// DefaultQualityThresholds returns the default quality thresholds
func DefaultQualityThresholds() QualityThresholds {
	return QualityThresholds{
		MinLaplacianVariance:       100.0,  // Minimum variance for sharpness (based on research)
		MaxLaplacianVariance:       2000.0, // Maximum variance to detect over-sharpening/noise
		MinLaplacianVarianceForOCR: 500.0,  // Higher threshold for OCR quality
		MinBrightness:              80.0,
		MaxBrightness:              220.0,
		MinLuminance:               0.2,
		MaxLuminance:               0.9,
		MinSaturation:              0.05,
		MaxChannelImbalance:        0.15,
		MaxSkewAngle:               5.0,
		MinWidth:                   800,
		MinHeight:                  1000,
		MinTotalPixels:             800000,
	}
}

// QualityValidator handles image quality validation logic
type QualityValidator struct {
	thresholds QualityThresholds
}

// NewQualityValidator creates a new quality validator with default thresholds
func NewQualityValidator() *QualityValidator {
	return &QualityValidator{
		thresholds: DefaultQualityThresholds(),
	}
}

// NewQualityValidatorWithThresholds creates a quality validator with custom thresholds
func NewQualityValidatorWithThresholds(thresholds QualityThresholds) *QualityValidator {
	return &QualityValidator{
		thresholds: thresholds,
	}
}

// QualityIssue represents a quality validation issue
type QualityIssue struct {
	Type        string  `json:"type"`
	Message     string  `json:"message"`
	Severity    string  `json:"severity"` // "error", "warning", "info"
	ActualValue float64 `json:"actual_value,omitempty"`
	Threshold   float64 `json:"threshold,omitempty"`
}

// ImageQualityMetrics represents the metrics needed for quality validation
type ImageQualityMetrics struct {
	// Basic metrics
	Width          int
	Height         int
	LaplacianVar   float64
	Brightness     float64
	AvgLuminance   float64
	AvgSaturation  float64
	ChannelBalance [3]float64

	// Quality flags
	Overexposed      bool
	Oversaturated    bool
	IncorrectWB      bool
	IsTooDark        bool
	IsTooBright      bool
	IsSkewed         bool
	HasDocumentEdges bool

	// Optional metrics
	SkewAngle *float64
}

// isImageBlurry performs enhanced blur detection that considers image content
func (qv *QualityValidator) isImageBlurry(metrics ImageQualityMetrics) bool {

	// For images with very low Laplacian variance, check if other quality indicators suggest the image is actually clear
	if metrics.LaplacianVar <= qv.thresholds.MinLaplacianVariance {
		// If variance is extremely low (< 1), it's likely truly blurry
		if metrics.LaplacianVar < 1.0 {
			return true
		}

		// For moderate low variance (10-350), check other quality indicators
		// If luminance is reasonable and channels are balanced,
		// the image might just have uniform content areas
		isBalanced := qv.isChannelBalanced(metrics.ChannelBalance)
		luminanceOK := metrics.AvgLuminance >= 0.3 && metrics.AvgLuminance <= 0.8

		// Enhanced blur detection: Check if other quality indicators suggest the image is actually clear
		if luminanceOK && isBalanced && !metrics.Overexposed && !metrics.Oversaturated {
			// Image has good quality indicators despite low variance
			// Likely uniform content (e.g., organized shelves, clean surfaces)
			// Removed saturation requirement as uniform images naturally have lower saturation
			return false
		}

		// Otherwise, consider it blurry
		return true
	}

	// Normal variance range - not blurry
	return false
}

// isChannelBalanced checks if RGB channels are reasonably balanced
func (qv *QualityValidator) isChannelBalanced(channels [3]float64) bool {
	max := math.Max(channels[0], math.Max(channels[1], channels[2]))
	min := math.Min(channels[0], math.Min(channels[1], channels[2]))
	return (max - min) <= qv.thresholds.MaxChannelImbalance
}

// ValidateBasicQuality performs basic quality validation suitable for general image analysis
func (qv *QualityValidator) ValidateBasicQuality(metrics ImageQualityMetrics) []QualityIssue {
	var issues []QualityIssue

	// 1. Blurriness (Laplacian Variance) - Enhanced detection
	if qv.isImageBlurry(metrics) {
		issues = append(issues, QualityIssue{
			Type:        "blurriness",
			Message:     "Image is blurry. Please hold the camera steady and try again.",
			Severity:    "error",
			ActualValue: metrics.LaplacianVar,
			Threshold:   qv.thresholds.MinLaplacianVariance,
		})
	} else if metrics.LaplacianVar >= qv.thresholds.MaxLaplacianVariance {
		issues = append(issues, QualityIssue{
			Type:        "over_sharpening",
			Message:     "Image has too much noise or artificial sharpening. Use natural lighting and avoid digital zoom.",
			Severity:    "error",
			ActualValue: metrics.LaplacianVar,
			Threshold:   qv.thresholds.MaxLaplacianVariance,
		})
	}

	// 2. Overexposure / Oversaturation
	if metrics.Overexposed {
		issues = append(issues, QualityIssue{
			Type:     "overexposure",
			Message:  "Image has too much light. Move to a less bright area.",
			Severity: "error",
		})
	}
	if metrics.Oversaturated {
		issues = append(issues, QualityIssue{
			Type:     "oversaturation",
			Message:  "Colors are too strong. Use normal light while clicking.",
			Severity: "error",
		})
	}

	// 3. White Balance
	if metrics.IncorrectWB {
		issues = append(issues, QualityIssue{
			Type:     "white_balance",
			Message:  "Colors in the photo don't look natural. Use normal lighting.",
			Severity: "warning",
		})
	}

	// 4. Average Luminance
	if metrics.AvgLuminance <= qv.thresholds.MinLuminance {
		issues = append(issues, QualityIssue{
			Type:        "low_luminance",
			Message:     "Image is very dull. Use more light.",
			Severity:    "error",
			ActualValue: metrics.AvgLuminance,
			Threshold:   qv.thresholds.MinLuminance,
		})
	} else if metrics.AvgLuminance >= qv.thresholds.MaxLuminance {
		issues = append(issues, QualityIssue{
			Type:        "high_luminance",
			Message:     "Image is too bright. Take it in normal light.",
			Severity:    "error",
			ActualValue: metrics.AvgLuminance,
			Threshold:   qv.thresholds.MaxLuminance,
		})
	}

	// 5. Saturation
	if metrics.AvgSaturation <= qv.thresholds.MinSaturation {
		issues = append(issues, QualityIssue{
			Type:        "low_saturation",
			Message:     "Image looks faded. Use proper lighting.",
			Severity:    "warning",
			ActualValue: metrics.AvgSaturation,
			Threshold:   qv.thresholds.MinSaturation,
		})
	}

	// 6. Channel Balance
	channels := metrics.ChannelBalance
	if math.Abs(channels[0]-channels[1]) >= qv.thresholds.MaxChannelImbalance ||
		math.Abs(channels[0]-channels[2]) >= qv.thresholds.MaxChannelImbalance ||
		math.Abs(channels[1]-channels[2]) >= qv.thresholds.MaxChannelImbalance {
		issues = append(issues, QualityIssue{
			Type:      "channel_imbalance",
			Message:   "Colors look odd. Don't use filters or colored lights.",
			Severity:  "warning",
			Threshold: qv.thresholds.MaxChannelImbalance,
		})
	}

	return issues
}

// ValidateOCRQuality performs comprehensive quality validation suitable for OCR analysis
func (qv *QualityValidator) ValidateOCRQuality(metrics ImageQualityMetrics) []QualityIssue {
	var issues []QualityIssue

	// Start with basic quality validation
	issues = qv.ValidateBasicQuality(metrics)

	// Override blurriness threshold for OCR (more strict)
	var foundBlurrinessIssue bool
	for i := len(issues) - 1; i >= 0; i-- {
		if issues[i].Type == "blurriness" {
			foundBlurrinessIssue = true
			if metrics.LaplacianVar <= qv.thresholds.MinLaplacianVarianceForOCR {
				issues[i].Threshold = qv.thresholds.MinLaplacianVarianceForOCR
			} else {
				// Remove the issue if it passes OCR threshold
				issues = append(issues[:i], issues[i+1:]...)
			}
			break
		}
		// Keep over_sharpening issues for OCR as they're still problematic
	}

	// If no existing blurriness issue was found but LaplacianVar is between thresholds, create new issue
	if !foundBlurrinessIssue && metrics.LaplacianVar <= qv.thresholds.MinLaplacianVarianceForOCR && metrics.LaplacianVar > qv.thresholds.MinLaplacianVariance {
		issues = append(issues, QualityIssue{
			Type:        "blurriness",
			Message:     "Image is blurry for OCR analysis. Please hold the camera steady and try again.",
			Severity:    "error",
			ActualValue: metrics.LaplacianVar,
			Threshold:   qv.thresholds.MinLaplacianVarianceForOCR,
		})
	}

	// Add OCR-specific validations

	// 1. Resolution check
	totalPixels := metrics.Width * metrics.Height
	if totalPixels < qv.thresholds.MinTotalPixels ||
		metrics.Width < qv.thresholds.MinWidth ||
		metrics.Height < qv.thresholds.MinHeight {
		issues = append(issues, QualityIssue{
			Type:        "low_resolution",
			Message:     "Image is too small or unclear. Please take a clearer photo.",
			Severity:    "error",
			ActualValue: float64(totalPixels),
			Threshold:   float64(qv.thresholds.MinTotalPixels),
		})
	}

	// 2. Brightness (more specific for OCR)
	if metrics.IsTooDark {
		issues = append(issues, QualityIssue{
			Type:        "too_dark",
			Message:     "Image is too dark. Take the photo in more light.",
			Severity:    "error",
			ActualValue: metrics.Brightness,
			Threshold:   qv.thresholds.MinBrightness,
		})
	}
	if metrics.IsTooBright {
		issues = append(issues, QualityIssue{
			Type:        "too_bright",
			Message:     "Image is too bright. Avoid strong sunlight or flash.",
			Severity:    "error",
			ActualValue: metrics.Brightness,
			Threshold:   qv.thresholds.MaxBrightness,
		})
	}

	// 3. Skew check
	if metrics.SkewAngle != nil && math.Abs(*metrics.SkewAngle) > qv.thresholds.MaxSkewAngle {
		issues = append(issues, QualityIssue{
			Type:        "skew",
			Message:     "Image is tilted. Hold the phone straight while clicking.",
			Severity:    "warning",
			ActualValue: math.Abs(*metrics.SkewAngle),
			Threshold:   qv.thresholds.MaxSkewAngle,
		})
	}

	// 4. Document edges check
	if !metrics.HasDocumentEdges {
		issues = append(issues, QualityIssue{
			Type:     "document_edges",
			Message:  "Full paper is not visible. Make sure all corners are inside the photo.",
			Severity: "error",
		})
	}

	return issues
}

// ConvertIssuesToMessages converts quality issues to simple error messages for backward compatibility
func (qv *QualityValidator) ConvertIssuesToMessages(issues []QualityIssue) []string {
	var messages []string
	for _, issue := range issues {
		messages = append(messages, issue.Message)
	}
	return messages
}

// HasCriticalIssues checks if there are any critical (error severity) issues
func (qv *QualityValidator) HasCriticalIssues(issues []QualityIssue) bool {
	for _, issue := range issues {
		if issue.Severity == "error" {
			return true
		}
	}
	return false
}
