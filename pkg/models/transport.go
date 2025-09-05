package models

// AnalysisRequest represents a request for image analysis
// Moved from transport package for shared usage
type AnalysisRequest struct {
	URL          string `json:"url" binding:"required,url"`
	IsOCR        bool   `json:"is_ocr,omitempty"`
	ExpectedText string `json:"expected_text,omitempty"`
}

// AnalysisOptionsRequest represents a request for image analysis with options
// Moved from transport package for shared usage
// Note: Options field uses interface{} to avoid import cycle with analyzer package
type AnalysisOptionsRequest struct {
	URL     string      `json:"url" binding:"required,url"`
	Options interface{} `json:"options,omitempty"`
}

// ErrorResponse represents an error response
// Moved from transport package for shared usage
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// ImageAnalysisResponse represents the response from image analysis
// Consolidates response structs from service and transport layers
type ImageAnalysisResponse struct {
	ImageURL          string       `json:"image_url"`
	Timestamp         string       `json:"timestamp"`
	ProcessingTimeSec float64      `json:"processing_time_sec"`
	Quality           Quality      `json:"quality"`
	Metrics           ImageMetrics `json:"metrics"`
	OCRResult         *OCRResult   `json:"ocr_result,omitempty"`
	Errors            []string     `json:"errors,omitempty"`
}
