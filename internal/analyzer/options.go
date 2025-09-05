package analyzer

// AnalysisOptions provides flexible configuration for image analysis
type AnalysisOptions struct {
	// Analysis modes
	OCRMode     bool `json:"ocr_mode"`
	FastMode    bool `json:"fast_mode"`
	QualityMode bool `json:"quality_mode"`

	// Quality thresholds
	BlurThreshold           float64 `json:"blur_threshold"`
	OverexposureThreshold   float64 `json:"overexposure_threshold"`
	OversaturationThreshold float64 `json:"oversaturation_threshold"`
	LuminanceThreshold      float64 `json:"luminance_threshold"`

	// Feature toggles
	SkipQRDetection      bool `json:"skip_qr_detection"`
	SkipWhiteBalance     bool `json:"skip_white_balance"`
	SkipContourDetection bool `json:"skip_contour_detection"`
	SkipEdgeDetection    bool `json:"skip_edge_detection"`

	// OCR-specific options
	OCRExpectedText string `json:"expected_text"`
	OCRLanguage     string `json:"ocr_language"`
	OCREngineMode   string `json:"ocr_engine_mode"` // "fast", "accurate", "legacy"

	// Performance options
	UseWorkerPool bool `json:"use_worker_pool"`
	MaxWorkers    int  `json:"max_workers"`
}

// DefaultOptions returns default analysis options
func DefaultOptions() AnalysisOptions {
	return AnalysisOptions{
		OCRMode:                 false,
		FastMode:                false,
		QualityMode:             true,
		BlurThreshold:           100.0,
		OverexposureThreshold:   0.95,
		OversaturationThreshold: 0.9,
		LuminanceThreshold:      0.95,
		SkipQRDetection:         false,
		SkipWhiteBalance:        false,
		SkipContourDetection:    false,
		SkipEdgeDetection:       false,
		UseWorkerPool:           true,
		MaxWorkers:              0, // Use default CPU count
	}
}

// OCROptions returns options for OCR analysis
func OCROptions() AnalysisOptions {
	opts := DefaultOptions()
	opts.OCRMode = true
	opts.QualityMode = true
	opts.BlurThreshold = 300.0  // Stricter blur detection for OCR
	opts.SkipQRDetection = true // QR codes not relevant for OCR
	opts.OCRLanguage = "eng"
	opts.OCREngineMode = "accurate"
	return opts
}

// FastOptions returns options for fast analysis
func FastOptions() AnalysisOptions {
	opts := DefaultOptions()
	opts.FastMode = true
	opts.QualityMode = false
	opts.SkipContourDetection = true
	opts.SkipEdgeDetection = true
	opts.SkipWhiteBalance = true
	return opts
}

// QualityOptions returns options for quality analysis
func QualityOptions() AnalysisOptions {
	opts := DefaultOptions()
	opts.QualityMode = true
	opts.BlurThreshold = 400.0 // More sensitive blur detection
	opts.OverexposureThreshold = 0.9
	opts.OversaturationThreshold = 0.85
	return opts
}

// WithOCR returns options with OCR enabled and expected text
func (opts AnalysisOptions) WithOCR(expectedText string) AnalysisOptions {
	opts.OCRMode = true
	opts.OCRExpectedText = expectedText
	opts.QualityMode = true
	return opts
}

// WithCustomThresholds allows setting custom quality thresholds
func (opts AnalysisOptions) WithCustomThresholds(blur, overexposure, oversaturation float64) AnalysisOptions {
	opts.BlurThreshold = blur
	opts.OverexposureThreshold = overexposure
	opts.OversaturationThreshold = oversaturation
	return opts
}

// WithFastMode enables fast analysis mode
func (opts AnalysisOptions) WithFastMode() AnalysisOptions {
	opts.FastMode = true
	opts.QualityMode = false
	opts.SkipContourDetection = true
	opts.SkipEdgeDetection = true
	return opts
}

// WithoutQRDetection disables QR code detection
func (opts AnalysisOptions) WithoutQRDetection() AnalysisOptions {
	opts.SkipQRDetection = true
	return opts
}
