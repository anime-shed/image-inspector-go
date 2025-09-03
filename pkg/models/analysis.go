package models

import "time"

// AnalysisResult represents the complete result of image analysis
// This consolidates the duplicate AnalysisResult structs from analyzer and repository packages
type AnalysisResult struct {
	ID                string     `json:"id"`
	ImageURL          string     `json:"image_url"`
	Timestamp         time.Time  `json:"timestamp"`
	ProcessingTimeSec float64    `json:"processing_time_sec"`
	
	// Quality indicators
	Quality Quality `json:"quality"`
	
	// Metrics
	Metrics ImageMetrics `json:"metrics"`
	
	// OCR specific (optional)
	OCRResult *OCRResult `json:"ocr_result,omitempty"`
	
	// Validation errors
	Errors []string `json:"errors,omitempty"`
}

// Quality represents image quality assessment
// Consolidates quality-related fields from multiple structs
type Quality struct {
	Overexposed   bool `json:"overexposed"`
	Oversaturated bool `json:"oversaturated"`
	IncorrectWB   bool `json:"incorrect_white_balance"`
	Blurry        bool `json:"blurry"`
	IsValid       bool `json:"is_valid"`
	
	// Enhanced quality checks for OCR
	IsLowResolution bool     `json:"is_low_resolution,omitempty"`
	IsTooDark       bool     `json:"is_too_dark,omitempty"`
	IsTooBright     bool     `json:"is_too_bright,omitempty"`
	IsSkewed        bool     `json:"is_skewed,omitempty"`
	HasDocumentEdges bool    `json:"has_document_edges,omitempty"`
	QRDetected      bool     `json:"qr_detected,omitempty"`
	SkewAngle       *float64 `json:"skew_angle,omitempty"`
}

// ImageMetrics represents image analysis metrics
// Consolidates metrics from analyzer and service packages
type ImageMetrics struct {
	LaplacianVar      float64    `json:"laplacian_variance"`
	AvgLuminance      float64    `json:"average_luminance"`
	AvgSaturation     float64    `json:"average_saturation"`
	ChannelBalance    [3]float64 `json:"channel_balance"`
	Resolution        string     `json:"resolution,omitempty"`
	Brightness        float64    `json:"brightness,omitempty"`
	NumContours       int        `json:"num_contours,omitempty"`
}

// OCRResult represents OCR analysis results
// Consolidates OCR-related fields from multiple packages
type OCRResult struct {
	ExtractedText string  `json:"extracted_text"`
	ExpectedText  string  `json:"expected_text,omitempty"`
	Confidence    float64 `json:"confidence"`
	MatchScore    float64 `json:"match_score,omitempty"`
	
	// Error rates for quality assessment
	WER      float64 `json:"word_error_rate,omitempty"`
	CER      float64 `json:"character_error_rate,omitempty"`
	OCRError string  `json:"ocr_error,omitempty"`
}

// ImageMetadata contains metadata about an image
// Moved from repository package for shared usage
type ImageMetadata struct {
	ContentType   string `json:"content_type"`
	ContentLength int64  `json:"content_length"`
	Width         int    `json:"width"`
	Height        int    `json:"height"`
	Format        string `json:"format"`
}

// ValidationError represents a structured validation error
type ValidationError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}