package analyzer

import "image"

// ImageAnalyzer defines the main interface for image analysis
type ImageAnalyzer interface {
	// Legacy methods for backward compatibility
	Analyze(img image.Image, isOCR bool) AnalysisResult
	AnalyzeWithOCR(img image.Image, expectedText string) AnalysisResult
	
	// New options-based method
	AnalyzeWithOptions(img image.Image, options AnalysisOptions) AnalysisResult
	
	// Lifecycle management
	Close() error
}

// MetricsCalculator handles image metrics computation
type MetricsCalculator interface {
	CalculateBasicMetrics(img image.Image) metrics
	CalculateLaplacianVariance(gray *image.Gray) float64
	CalculateBrightness(gray *image.Gray) float64
	DetectSkew(gray *image.Gray) *float64
	DetectContours(gray *image.Gray) int
}

// QRDetector handles QR code detection
type QRDetector interface {
	DetectQRCode(img image.Image) bool
}

// OCRAnalyzer handles OCR-specific analysis
type OCRAnalyzer interface {
	PerformOCRAnalysis(img image.Image, expectedText string) AnalysisResult
}