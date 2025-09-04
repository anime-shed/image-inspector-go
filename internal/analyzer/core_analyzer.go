package analyzer

import (
	"fmt"
	"go-image-inspector/pkg/models"
	"go-image-inspector/pkg/validation"
	"image"
	"image/draw"
	"strconv"
	"strings"
	"sync"
	"time"
)

// coreAnalyzer implements ImageAnalyzer interface with enhanced performance
// Implements optimizations from PERFORMANCE_OPTIMIZATION_ANALYSIS.md Phase 3
type coreAnalyzer struct {
	workerPool        *WorkerPool
	metricsCalculator MetricsCalculator
	qualityValidator  *validation.QualityValidator
	qrDetector        QRDetector

	// Enhanced memory pools with better sizing
	grayPool      sync.Pool
	resultPool    sync.Pool
	bufferPool    sync.Pool
	tempSlicePool sync.Pool

	// Performance monitoring
	analysisCount    int64
	totalProcessTime time.Duration
	mu               sync.RWMutex
}

// NewCoreAnalyzer creates a new core analyzer
func NewCoreAnalyzer() (ImageAnalyzer, error) {
	workerPool := NewWorkerPool(0) // Use default CPU count
	workerPool.Start()

	return &coreAnalyzer{
		workerPool:        workerPool,
		metricsCalculator: NewMetricsCalculator(),
		qualityValidator:  validation.NewQualityValidator(),
		qrDetector:        NewQRDetector(),

		// Enhanced memory pools
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
		bufferPool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, 8192) // 8KB buffer
			},
		},
		tempSlicePool: sync.Pool{
			New: func() interface{} {
				return make([]float64, 0, 2048) // 2K float64 elements
			},
		},
	}, nil
}

// Analyze performs basic image analysis with memory management
func (oca *coreAnalyzer) Analyze(img image.Image, isOCR bool) AnalysisResult {
	options := DefaultOptions()
	options.OCRMode = isOCR
	return oca.AnalyzeWithOptions(img, options)
}

// AnalyzeWithOCR performs OCR-specific image analysis (legacy method for backward compatibility)
func (oca *coreAnalyzer) AnalyzeWithOCR(img image.Image, expectedText string) AnalysisResult {
	options := OCROptions().WithOCR(expectedText)
	return oca.AnalyzeWithOptions(img, options)
}

// AnalyzeWithOptions performs image analysis with enhanced parallel processing and memory optimization
func (oca *coreAnalyzer) AnalyzeWithOptions(img image.Image, options AnalysisOptions) AnalysisResult {
	start := time.Now()
	defer func() {
		oca.updatePerformanceStats(time.Since(start))
	}()

	// Get result from pool and reset it efficiently
	result := oca.resultPool.Get().(*AnalysisResult)
	*result = AnalysisResult{} // Reset the result
	defer oca.resultPool.Put(result)

	result.Timestamp = start

	// Set expected text in OCR result if provided
	if options.OCRExpectedText != "" {
		if result.OCRResult == nil {
			result.OCRResult = &models.OCRResult{}
		}
		result.OCRResult.ExpectedText = options.OCRExpectedText
		// OCR is not implemented yet, set error message
		result.OCRResult.OCRError = "OCR text extraction is not implemented in this version"
	}

	// Grayscale conversion with memory reuse
	bounds := img.Bounds()
	gray := oca.getGrayImage(bounds)
	defer oca.grayPool.Put(gray)

	draw.Draw(gray, bounds, img, bounds.Min, draw.Src)

	// Parallel processing of different analysis components
	if options.UseWorkerPool && !options.FastMode {
		oca.analyzeWithParallelProcessing(img, gray, result, options)
	} else {
		oca.analyzeSequentially(img, gray, result, options)
	}

	// Calculate processing time
	processingTime := time.Since(start).Seconds()
	result.ProcessingTimeSec = processingTime

	// Create a copy to return
	finalResult := *result
	// Ensure processing time is copied
	finalResult.ProcessingTimeSec = processingTime
	return finalResult
}

// getGrayImage retrieves and properly sizes a grayscale image from the pool
func (oca *coreAnalyzer) getGrayImage(bounds image.Rectangle) *image.Gray {
	gray := oca.grayPool.Get().(*image.Gray)

	w, h := bounds.Dx(), bounds.Dy()
	requiredLen := w * h

	// Optimize memory allocation - reuse if possible, allocate if needed
	if cap(gray.Pix) < requiredLen {
		// Need more capacity
		gray.Pix = make([]uint8, requiredLen)
	} else {
		// Reuse existing capacity
		gray.Pix = gray.Pix[:requiredLen]
	}

	gray.Stride = w
	gray.Rect = bounds

	return gray
}

// analyzeWithParallelProcessing performs analysis using parallel worker pool
func (oca *coreAnalyzer) analyzeWithParallelProcessing(img image.Image, gray *image.Gray, result *AnalysisResult, options AnalysisOptions) {
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Set resolution information (needed for quality validation)
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	result.Metrics.Resolution = fmt.Sprintf("%dx%d", width, height)

	// Basic metrics calculation
	wg.Add(1)
	oca.workerPool.Submit(func() {
		defer wg.Done()
		metrics := oca.metricsCalculator.CalculateBasicMetrics(img)

		mu.Lock()
		result.Metrics.AvgLuminance = metrics.avgLuminance
		result.Metrics.AvgSaturation = metrics.avgSaturation
		result.Metrics.ChannelBalance = [3]float64{metrics.avgR, metrics.avgG, metrics.avgB}
		mu.Unlock()
	})

	// Laplacian variance calculation
	wg.Add(1)
	oca.workerPool.Submit(func() {
		defer wg.Done()
		laplacianVar := oca.metricsCalculator.CalculateLaplacianVariance(gray)

		mu.Lock()
		result.Metrics.LaplacianVar = laplacianVar
		result.Quality.Blurry = laplacianVar <= options.BlurThreshold
		mu.Unlock()
	})

	// QR detection (if enabled)
	if !options.SkipQRDetection {
		wg.Add(1)
		oca.workerPool.Submit(func() {
			defer wg.Done()
			qrDetected := oca.qrDetector.DetectQRCode(img)

			mu.Lock()
			result.Quality.QRDetected = qrDetected
			mu.Unlock()
		})
	}

	// Enhanced quality checks for OCR mode
	if options.OCRMode {
		wg.Add(1)
		oca.workerPool.Submit(func() {
			defer wg.Done()
			oca.performEnhancedQualityChecks(img, gray, result, options)
		})
	}

	wg.Wait()

	// Perform quality validation to populate error messages
	oca.performQualityValidation(result, options)

	// Post-process results
	oca.finalizeAnalysisResults(result, options)
}

// analyzeSequentially performs analysis without parallel processing (for fast mode)
func (oca *coreAnalyzer) analyzeSequentially(img image.Image, gray *image.Gray, result *AnalysisResult, options AnalysisOptions) {
	// Calculate basic metrics
	metrics := oca.metricsCalculator.CalculateBasicMetrics(img)
	result.Metrics.AvgLuminance = metrics.avgLuminance
	result.Metrics.AvgSaturation = metrics.avgSaturation
	result.Metrics.ChannelBalance = [3]float64{metrics.avgR, metrics.avgG, metrics.avgB}

	// Calculate Laplacian variance for blur detection
	result.Metrics.LaplacianVar = oca.metricsCalculator.CalculateLaplacianVariance(gray)
	result.Quality.Blurry = result.Metrics.LaplacianVar <= options.BlurThreshold

	// Set resolution information (needed for quality validation)
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	result.Metrics.Resolution = fmt.Sprintf("%dx%d", width, height)

	// Check for overexposure and oversaturation
	result.Quality.Overexposed = metrics.avgLuminance > options.OverexposureThreshold
	result.Quality.Oversaturated = metrics.avgSaturation > options.OversaturationThreshold

	// Check white balance (skip if disabled)
	if !options.SkipWhiteBalance {
		result.Quality.IncorrectWB = oca.hasWhiteBalanceIssue(metrics.avgR, metrics.avgG, metrics.avgB)
	}

	// Detect QR codes (skip if disabled)
	if !options.SkipQRDetection {
		result.Quality.QRDetected = oca.qrDetector.DetectQRCode(img)
	}

	// Enhanced quality checks for OCR mode
	if options.OCRMode {
		oca.performEnhancedQualityChecks(img, gray, result, options)
	}

	// Perform quality validation to populate error messages
	oca.performQualityValidation(result, options)

	oca.finalizeAnalysisResults(result, options)
}

// performEnhancedQualityChecks performs additional quality checks for OCR with optimizations
func (oca *coreAnalyzer) performEnhancedQualityChecks(img image.Image, gray *image.Gray, result *AnalysisResult, options AnalysisOptions) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Set resolution information
	result.Metrics.Resolution = fmt.Sprintf("%dx%d", width, height)
	result.Quality.IsLowResolution = width*height < 800000 || width < 800 || height < 1000

	// Calculate brightness
	result.Metrics.Brightness = oca.metricsCalculator.CalculateBrightness(gray)
	result.Quality.IsTooDark = result.Metrics.Brightness < 80
	result.Quality.IsTooBright = result.Metrics.Brightness > 220

	// Detect skew
	skewAngle := oca.metricsCalculator.DetectSkew(gray)
	if skewAngle != nil {
		result.Quality.SkewAngle = skewAngle
		result.Quality.IsSkewed = *skewAngle > 5 || *skewAngle < -5
	}

	// Count contours (skip if disabled)
	if !options.SkipContourDetection {
		result.Metrics.NumContours = oca.metricsCalculator.DetectContours(gray)
	}

	// Simple document edge detection (skip if disabled)
	if !options.SkipEdgeDetection {
		result.Quality.HasDocumentEdges = oca.detectDocumentEdges(gray)
	}

	// Perform quality validation using QualityValidator
	oca.performQualityValidation(result, options)
}

// performQualityValidation uses QualityValidator to generate quality error messages
func (oca *coreAnalyzer) performQualityValidation(result *AnalysisResult, options AnalysisOptions) {
	// Prepare metrics for validation
	width := oca.getWidthFromResolution(result.Metrics.Resolution)
	height := oca.getHeightFromResolution(result.Metrics.Resolution)
	
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

	// Perform appropriate validation based on mode
	var issues []validation.QualityIssue
	if options.OCRMode {
		issues = oca.qualityValidator.ValidateOCRQuality(metrics)
	} else {
		issues = oca.qualityValidator.ValidateBasicQuality(metrics)
	}

	// Convert issues to error messages
	if len(issues) > 0 {
		result.Errors = oca.qualityValidator.ConvertIssuesToMessages(issues)
	}
}

// getWidthFromResolution extracts width from resolution string
func (oca *coreAnalyzer) getWidthFromResolution(resolution string) int {
	width, _ := oca.parseResolution(resolution)
	return width
}

// getHeightFromResolution extracts height from resolution string
func (oca *coreAnalyzer) getHeightFromResolution(resolution string) int {
	_, height := oca.parseResolution(resolution)
	return height
}

// finalizeAnalysisResults performs final processing and validation
func (oca *coreAnalyzer) finalizeAnalysisResults(result *AnalysisResult, options AnalysisOptions) {
	// Set overall validity based on quality checks and validation errors
	hasQualityIssues := result.Quality.Blurry ||
		result.Quality.Overexposed ||
		result.Quality.Oversaturated ||
		(options.OCRMode && (result.Quality.IsTooDark || result.Quality.IsTooBright))
	
	// Also consider validation errors from QualityValidator
	hasValidationErrors := len(result.Errors) > 0
	
	// Image is valid only if it has no quality issues AND no validation errors
	result.Quality.IsValid = !hasQualityIssues && !hasValidationErrors
}

// hasWhiteBalanceIssue checks for white balance issues
func (oca *coreAnalyzer) hasWhiteBalanceIssue(avgR, avgG, avgB float64) bool {
	threshold := 0.15
	maxChannel := maxFloat64(avgR, maxFloat64(avgG, avgB))
	minChannel := minFloat64(avgR, minFloat64(avgG, avgB))
	return (maxChannel - minChannel) > threshold
}

// detectDocumentEdges performs basic document edge detection
func (oca *coreAnalyzer) detectDocumentEdges(gray *image.Gray) bool {
	bounds := gray.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Simple heuristic: check if corners are significantly different from center
	corners := []image.Point{
		{bounds.Min.X + 10, bounds.Min.Y + 10}, // Top-left
		{bounds.Max.X - 10, bounds.Min.Y + 10}, // Top-right
		{bounds.Min.X + 10, bounds.Max.Y - 10}, // Bottom-left
		{bounds.Max.X - 10, bounds.Max.Y - 10}, // Bottom-right
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

	return differentCorners >= 2
}

// updatePerformanceStats updates internal performance statistics
func (oca *coreAnalyzer) updatePerformanceStats(duration time.Duration) {
	oca.mu.Lock()
	defer oca.mu.Unlock()

	oca.analysisCount++
	oca.totalProcessTime += duration
}

// GetPerformanceStats returns current performance statistics
func (oca *coreAnalyzer) GetPerformanceStats() (int64, time.Duration) {
	oca.mu.RLock()
	defer oca.mu.RUnlock()

	return oca.analysisCount, oca.totalProcessTime
}

// Helper functions
func maxFloat64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minFloat64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// abs returns the absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// parseResolution parses the resolution string (e.g., "1920x1080") and returns width and height
func (oca *coreAnalyzer) parseResolution(resolution string) (int, int) {
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
