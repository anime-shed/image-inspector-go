package analyzer

// AnalysisOptions provides flexible configuration for image analysis
type AnalysisOptions struct {
	// Analysis modes
	OCRMode     bool
	FastMode    bool
	QualityMode bool

	// Quality thresholds
	BlurThreshold         float64
	OverexposureThreshold float64
	OversaturationThreshold float64
	LuminanceThreshold    float64

	// Feature toggles
	SkipQRDetection      bool
	SkipWhiteBalance     bool
	SkipContourDetection bool
	SkipEdgeDetection    bool

	// OCR-specific options
	OCRExpectedText string
	OCRLanguage     string
	OCREngineMode   string // "fast", "accurate", "legacy"

	// Performance options
	UseWorkerPool bool
	MaxWorkers    int
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
	opts.BlurThreshold = 300.0 // Stricter blur detection for OCR
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