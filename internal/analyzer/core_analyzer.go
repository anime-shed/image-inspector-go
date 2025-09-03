package analyzer

import (
	"fmt"
	"image"
	"image/draw"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"
	"go-image-inspector/pkg/models"
	"go-image-inspector/pkg/validation"
)

// coreAnalyzer implements ImageAnalyzer interface and orchestrates all components
type coreAnalyzer struct {
	workerPool        *WorkerPool
	metricsCalculator MetricsCalculator
	qualityValidator  *validation.QualityValidator
	qrDetector        QRDetector
	grayPool          sync.Pool
	resultPool        sync.Pool
}

// NewImageAnalyzer creates a new image analyzer with all components
func NewImageAnalyzer() (ImageAnalyzer, error) {
	workerPool := NewWorkerPool(0) // Use default CPU count
	workerPool.Start()

	return &coreAnalyzer{
		workerPool:        workerPool,
		metricsCalculator: NewMetricsCalculator(),
		qualityValidator:  validation.NewQualityValidator(),
		qrDetector:        NewQRDetector(),
		grayPool: sync.Pool{
			New: func() interface{} {
				return &image.Gray{}
			},
		},
		resultPool: sync.Pool{
			New: func() interface{} {
				return &AnalysisResult{}
			},
		},
	}, nil
}

// Analyze performs basic image analysis (legacy method for backward compatibility)
func (ca *coreAnalyzer) Analyze(img image.Image, isOCR bool) AnalysisResult {
	options := DefaultOptions()
	options.OCRMode = isOCR
	return ca.AnalyzeWithOptions(img, options)
}

// AnalyzeWithOptions performs image analysis with flexible configuration
func (ca *coreAnalyzer) AnalyzeWithOptions(img image.Image, options AnalysisOptions) AnalysisResult {
	start := time.Now()

	// Get result from pool and reset it
	result := ca.resultPool.Get().(*AnalysisResult)
	*result = AnalysisResult{} // Reset the result
	defer ca.resultPool.Put(result)

	result.Timestamp = start
	// Set expected text in OCR result if provided
	if options.OCRExpectedText != "" {
		if result.OCRResult == nil {
			result.OCRResult = &models.OCRResult{}
		}
		result.OCRResult.ExpectedText = options.OCRExpectedText
	}

	// Convert to grayscale for analysis
	bounds := img.Bounds()
	gray := ca.grayPool.Get().(*image.Gray)
	gray.Pix = nil
	gray.Stride = 0
	gray.Rect = image.Rectangle{}
	defer ca.grayPool.Put(gray)

	*gray = *image.NewGray(bounds)
	draw.Draw(gray, bounds, img, bounds.Min, draw.Src)

	// Calculate basic metrics
	metrics := ca.metricsCalculator.CalculateBasicMetrics(img)
	result.Metrics.AvgLuminance = metrics.avgLuminance
	result.Metrics.AvgSaturation = metrics.avgSaturation
	result.Metrics.ChannelBalance = [3]float64{metrics.avgR, metrics.avgG, metrics.avgB}

	// Calculate Laplacian variance for blur detection using configurable threshold
	result.Metrics.LaplacianVar = ca.metricsCalculator.CalculateLaplacianVariance(gray)
	result.Quality.Blurry = result.Metrics.LaplacianVar <= options.BlurThreshold

	// Check for overexposure and oversaturation using configurable thresholds
	result.Quality.Overexposed = metrics.avgLuminance > options.OverexposureThreshold
	result.Quality.Oversaturated = metrics.avgSaturation > options.OversaturationThreshold

	// Check white balance (skip if disabled)
	if !options.SkipWhiteBalance {
		result.Quality.IncorrectWB = ca.hasWhiteBalanceIssue(metrics.avgR, metrics.avgG, metrics.avgB)
	}

	// Detect QR codes (skip if disabled)
	if !options.SkipQRDetection {
		result.Quality.QRDetected = ca.qrDetector.DetectQRCode(img)
	}

	// Perform enhanced quality checks if OCR analysis is requested or quality mode is enabled
	if options.OCRMode || options.QualityMode {
		if !options.FastMode {
			ca.performEnhancedQualityChecks(img, gray, result, options)
		}
		ca.validateComprehensiveQuality(result)
	} else {
		ca.validateBasicQuality(result)
	}

	// Handle OCR processing if enabled
	if options.OCRMode {
		// TODO: Implement actual OCR processing
		// This would integrate with an OCR library like Tesseract
		if result.OCRResult == nil {
			result.OCRResult = &models.OCRResult{}
		}
		result.OCRResult.ExtractedText = "" // Placeholder
		result.OCRResult.OCRError = "OCR processing not implemented yet"
		// TODO: Calculate WER and CER when OCR is implemented
		result.OCRResult.WER = 0.0
		result.OCRResult.CER = 0.0
	}

	result.ProcessingTimeSec = time.Since(start).Seconds()

	// Return a copy of the result
	finalResult := *result
	return finalResult
}

// AnalyzeWithOCR performs OCR-specific image analysis
// AnalyzeWithOCR performs OCR analysis (legacy method for backward compatibility)
func (ca *coreAnalyzer) AnalyzeWithOCR(img image.Image, expectedText string) AnalysisResult {
	options := OCROptions().WithOCR(expectedText)
	return ca.AnalyzeWithOptions(img, options)
}

// performEnhancedQualityChecks performs additional quality checks for OCR
func (ca *coreAnalyzer) performEnhancedQualityChecks(img image.Image, gray *image.Gray, result *AnalysisResult, options AnalysisOptions) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Set resolution information
	result.Metrics.Resolution = fmt.Sprintf("%dx%d", width, height)
	result.Quality.IsLowResolution = width*height < 800000 || width < 800 || height < 1000

	// Calculate brightness
	result.Metrics.Brightness = ca.metricsCalculator.CalculateBrightness(gray)
	result.Quality.IsTooDark = result.Metrics.Brightness < 80
	result.Quality.IsTooBright = result.Metrics.Brightness > 220

	// Detect skew
	skewAngle := ca.metricsCalculator.DetectSkew(gray)
	if skewAngle != nil {
		result.Quality.SkewAngle = skewAngle
		result.Quality.IsSkewed = *skewAngle > 5 || *skewAngle < -5
	}

	// Count contours (skip if disabled)
	if !options.SkipContourDetection {
		result.Metrics.NumContours = ca.metricsCalculator.DetectContours(gray)
	}

	// Simple document edge detection (skip if disabled)
	if !options.SkipEdgeDetection {
		// This would be more sophisticated in a real implementation
		result.Quality.HasDocumentEdges = ca.detectDocumentEdges(gray)
	}
}

// detectDocumentEdges performs basic document edge detection
func (ca *coreAnalyzer) detectDocumentEdges(gray *image.Gray) bool {
	bounds := gray.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Simple heuristic: check if corners are significantly different from center
	corners := []image.Point{
		{bounds.Min.X + 10, bounds.Min.Y + 10},         // Top-left
		{bounds.Max.X - 10, bounds.Min.Y + 10},         // Top-right
		{bounds.Min.X + 10, bounds.Max.Y - 10},         // Bottom-left
		{bounds.Max.X - 10, bounds.Max.Y - 10},         // Bottom-right
	}

	center := gray.GrayAt(width/2, height/2).Y
	differentCorners := 0

	for _, corner := range corners {
		if corner.X >= bounds.Min.X && corner.X < bounds.Max.X &&
			corner.Y >= bounds.Min.Y && corner.Y < bounds.Max.Y {
			cornerValue := gray.GrayAt(corner.X, corner.Y).Y
			if abs(int(cornerValue)-int(center)) > 30 {
				differentCorners++
			}
		}
	}

	// If at least 2 corners are significantly different, assume document edges exist
	return differentCorners >= 2
}

// hasWhiteBalanceIssue checks for white balance problems
func (ca *coreAnalyzer) hasWhiteBalanceIssue(avgR, avgG, avgB float64) bool {
	// Check if any channel is significantly different from the others
	threshold := 0.1
	maxDiff := math.Max(math.Abs(avgR-avgG), math.Max(math.Abs(avgR-avgB), math.Abs(avgG-avgB)))
	return maxDiff > threshold
}

// abs returns the absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// validateBasicQuality validates basic image quality conditions
func (ca *coreAnalyzer) validateBasicQuality(result *models.AnalysisResult) {
	// Parse width and height from resolution string
	width, height := ca.parseResolution(result.Metrics.Resolution)
	
	// Convert AnalysisResult to validation.ImageQualityMetrics
	metrics := validation.ImageQualityMetrics{
		Width:            width,
		Height:           height,
		LaplacianVar:     result.Metrics.LaplacianVar,
		Brightness:       result.Metrics.Brightness,
		AvgLuminance:     result.Metrics.AvgLuminance,
		AvgSaturation:    result.Metrics.AvgSaturation,
		ChannelBalance:   result.Metrics.ChannelBalance,
		Overexposed:      result.Quality.Overexposed,
		Oversaturated:    result.Quality.Oversaturated,
		IncorrectWB:      result.Quality.IncorrectWB,
		IsTooDark:        result.Quality.IsTooDark,
		IsTooBright:      result.Quality.IsTooBright,
		IsSkewed:         result.Quality.IsSkewed,
		HasDocumentEdges: result.Quality.HasDocumentEdges,
		SkewAngle:        result.Quality.SkewAngle,
	}

	// Use the shared validation package
	issues := ca.qualityValidator.ValidateBasicQuality(metrics)
	result.Errors = ca.qualityValidator.ConvertIssuesToMessages(issues)
}

// validateComprehensiveQuality validates comprehensive image quality for OCR analysis
func (ca *coreAnalyzer) validateComprehensiveQuality(result *models.AnalysisResult) {
	// Parse width and height from resolution string
	width, height := ca.parseResolution(result.Metrics.Resolution)
	
	// Convert AnalysisResult to validation.ImageQualityMetrics
	metrics := validation.ImageQualityMetrics{
		Width:            width,
		Height:           height,
		LaplacianVar:     result.Metrics.LaplacianVar,
		Brightness:       result.Metrics.Brightness,
		AvgLuminance:     result.Metrics.AvgLuminance,
		AvgSaturation:    result.Metrics.AvgSaturation,
		ChannelBalance:   result.Metrics.ChannelBalance,
		Overexposed:      result.Quality.Overexposed,
		Oversaturated:    result.Quality.Oversaturated,
		IncorrectWB:      result.Quality.IncorrectWB,
		IsTooDark:        result.Quality.IsTooDark,
		IsTooBright:      result.Quality.IsTooBright,
		IsSkewed:         result.Quality.IsSkewed,
		HasDocumentEdges: result.Quality.HasDocumentEdges,
		SkewAngle:        result.Quality.SkewAngle,
	}

	// Use the shared validation package for OCR quality validation
	issues := ca.qualityValidator.ValidateOCRQuality(metrics)
	result.Errors = ca.qualityValidator.ConvertIssuesToMessages(issues)
}

// parseResolution parses the resolution string (e.g., "1920x1080") and returns width and height
func (ca *coreAnalyzer) parseResolution(resolution string) (int, int) {
	if resolution == "" {
		return 0, 0
	}
	
	parts := strings.Split(resolution, "x")
	if len(parts) != 2 {
		return 0, 0
	}
	
	width, err1 := strconv.Atoi(parts[0])
	height, err2 := strconv.Atoi(parts[1])
	
	if err1 != nil || err2 != nil {
		return 0, 0
	}
	
	return width, height
}