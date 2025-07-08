package analyzer

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"
	"strings"
	"time"

	"github.com/arbovm/levenshtein"
	"github.com/codycollier/wer"
	"github.com/otiai10/gosseract/v2"
)

type AnalysisResult struct {
	Overexposed    bool       `json:"overexposed"`
	Oversaturated  bool       `json:"oversaturated"`
	IncorrectWB    bool       `json:"incorrect_white_balance"`
	Blurry         bool       `json:"blurry"`
	LaplacianVar   float64    `json:"laplacian_variance"`
	AvgLuminance   float64    `json:"average_luminance"`
	AvgSaturation  float64    `json:"average_saturation"`
	ChannelBalance [3]float64 `json:"channel_balance"`

	// Enhanced quality checks (when isOCR=true)
	Resolution        string   `json:"resolution,omitempty"`
	IsLowResolution   bool     `json:"is_low_resolution,omitempty"`
	Brightness        float64  `json:"brightness,omitempty"`
	IsTooDark         bool     `json:"is_too_dark,omitempty"`
	IsTooBright       bool     `json:"is_too_bright,omitempty"`
	SkewAngle         *float64 `json:"skew_angle,omitempty"`
	IsSkewed          bool     `json:"is_skewed,omitempty"`
	NumContours       int      `json:"num_contours,omitempty"`
	HasDocumentEdges  bool     `json:"has_document_edges,omitempty"`
	TextCoverageRatio float64  `json:"text_coverage_ratio,omitempty"`
	LowTextCoverage   bool     `json:"low_text_coverage,omitempty"`
	AvgOCRConfidence  float64  `json:"avg_ocr_confidence,omitempty"`
	LowOCRConfidence  bool     `json:"low_ocr_confidence,omitempty"`
	QRDetected        bool     `json:"qr_detected,omitempty"`
	ProcessingTimeSec float64  `json:"processing_time_sec,omitempty"`

	// OCR related fields
	OCRText  string  `json:"ocr_text,omitempty"`
	WER      float64 `json:"word_error_rate,omitempty"`
	CER      float64 `json:"character_error_rate,omitempty"`
	OCRError string  `json:"ocr_error,omitempty"`
}

type ImageAnalyzer interface {
	Analyze(img image.Image, isOCR bool) AnalysisResult
	AnalyzeWithOCR(img image.Image, expectedText string) AnalysisResult
	Close() error
}

type imageAnalyzer struct {
	tesseractClient *gosseract.Client
}

func NewImageAnalyzer(lang string, psm gosseract.PageSegMode) (ImageAnalyzer, error) {
	client := gosseract.NewClient()
	// Set language
	err := client.SetLanguage(lang)
	if err != nil {
		client.Close() // Clean up if setting language fails
		return nil, err
	}
	// Set Page Segmentation Mode
	client.SetPageSegMode(psm)

	return &imageAnalyzer{
		tesseractClient: client,
	}, nil
}

// Close releases resources used by the analyzer
func (a *imageAnalyzer) Close() error {
	if a.tesseractClient != nil {
		return a.tesseractClient.Close()
	}
	return nil
}

func (a *imageAnalyzer) Analyze(img image.Image, isOCR bool) AnalysisResult {
	startTime := time.Now()
	bounds := img.Bounds()
	gray := image.NewGray(bounds)
	draw.Draw(gray, bounds, img, bounds.Min, draw.Src)

	metrics := a.calculateMetrics(img, bounds)
	variance := a.computeLaplacianVariance(gray)

	overexposedThreshold := 0.8
	oversaturatedThreshold := 0.7
	blurryThreshold := 150.0

	if isOCR {
		overexposedThreshold = 0.75
		oversaturatedThreshold = 0.65
		blurryThreshold = 200.0
	}

	result := AnalysisResult{
		Overexposed:    metrics.avgLuminance > overexposedThreshold || metrics.avgLuminance < 0.15,
		Oversaturated:  metrics.avgSaturation > oversaturatedThreshold,
		IncorrectWB:    a.hasWhiteBalanceIssue(metrics.avgR, metrics.avgG, metrics.avgB),
		Blurry:         variance < blurryThreshold,
		LaplacianVar:   variance,
		AvgLuminance:   metrics.avgLuminance,
		AvgSaturation:  metrics.avgSaturation,
		ChannelBalance: [3]float64{metrics.avgR, metrics.avgG, metrics.avgB},
	}

	// Enhanced quality checks when isOCR is true
	if isOCR {
		a.performEnhancedQualityChecks(img, gray, &result)
	}

	result.ProcessingTimeSec = time.Since(startTime).Seconds()
	return result
}

type metrics struct {
	avgLuminance, avgSaturation float64
	avgR, avgG, avgB            float64
}

func (a *imageAnalyzer) calculateMetrics(img image.Image, bounds image.Rectangle) metrics {
	var totalLum, totalSat, totalR, totalG, totalB float64
	pixelCount := float64(bounds.Dx() * bounds.Dy())

	type result struct {
		lum, sat, r, g, b float64
	}

	results := make(chan result, bounds.Dy())

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		go func(y int) {
			var lum, sat, r, g, b float64
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				rVal, gVal, bVal, _ := img.At(x, y).RGBA()
				rf, gf, bf := float64(rVal>>8), float64(gVal>>8), float64(bVal>>8)

				_, s, v := a.rgbToHSV(rf, gf, bf)
				sat += s
				lum += v

				r += rf
				g += gf
				b += bf
			}
			results <- result{lum, sat, r, g, b}
		}(y)
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		res := <-results
		totalLum += res.lum
		totalSat += res.sat
		totalR += res.r
		totalG += res.g
		totalB += res.b
	}

	return metrics{
		avgLuminance:  totalLum / pixelCount,
		avgSaturation: totalSat / pixelCount,
		avgR:          totalR / pixelCount,
		avgG:          totalG / pixelCount,
		avgB:          totalB / pixelCount,
	}
}

func (a *imageAnalyzer) rgbToHSV(r, g, b float64) (h, s, v float64) {
	max := math.Max(math.Max(r, g), b)
	min := math.Min(math.Min(r, g), b)
	delta := max - min

	v = max / 255
	if max == 0 {
		s = 0
	} else {
		s = delta / max
	}

	if delta == 0 {
		h = 0
	} else {
		switch max {
		case r:
			h = (g - b) / delta
		case g:
			h = 2 + (b-r)/delta
		case b:
			h = 4 + (r-g)/delta
		}
		h *= 60
		if h < 0 {
			h += 360
		}
	}
	return h, s, v
}

func (a *imageAnalyzer) computeLaplacianVariance(gray *image.Gray) float64 {
	bounds := gray.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

	var sum, sumSq float64
	kernel := [3][3]int{{0, 1, 0}, {1, -4, 1}, {0, 1, 0}}

	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			var val int
			for ky := -1; ky <= 1; ky++ {
				for kx := -1; kx <= 1; kx++ {
					pixel := int(gray.GrayAt(x+kx, y+ky).Y)
					val += pixel * kernel[ky+1][kx+1]
				}
			}
			fVal := float64(val)
			sum += fVal
			sumSq += fVal * fVal
		}
	}

	n := float64((width - 2) * (height - 2))
	if n == 0 {
		return 0
	}
	mean := sum / n
	return (sumSq / n) - (mean * mean)
}

func (a *imageAnalyzer) hasWhiteBalanceIssue(avgR, avgG, avgB float64) bool {
	avg := (avgR + avgG + avgB) / 3
	maxDeviation := 0.15 * avg // 15% tolerance
	return math.Abs(avgR-avg) > maxDeviation ||
		math.Abs(avgG-avg) > maxDeviation ||
		math.Abs(avgB-avg) > maxDeviation
}

// AnalyzeWithOCR performs image analysis and OCR processing with error metrics
func (a *imageAnalyzer) AnalyzeWithOCR(img image.Image, expectedText string) AnalysisResult {
	// First perform standard image analysis
	result := a.Analyze(img, true)

	// Convert image to bytes for OCR processing
	buf := new(bytes.Buffer)
	err := jpeg.Encode(buf, img, &jpeg.Options{Quality: 95})
	if err != nil {
		// Try PNG if JPEG encoding fails
		buf.Reset()
		err = png.Encode(buf, img)
		if err != nil {
			result.OCRError = "Failed to encode image for OCR processing (tried both JPEG and PNG): " + err.Error()
			return result
		}
	}

	// Perform OCR
	a.tesseractClient.SetImageFromBytes(buf.Bytes())
	ocrText, err := a.tesseractClient.Text()
	if err != nil {
		result.OCRError = "OCR processing failed: " + err.Error()
		return result
	}

	result.OCRText = ocrText

	// Calculate metrics if expected text is provided
	if expectedText != "" {
		expectedLower := strings.ToLower(expectedText)
		ocrLower := strings.ToLower(ocrText)
		expectedTokens := strings.Fields(expectedLower)
		ocrTokens := strings.Fields(ocrLower)

		werValue, _ := wer.WER(expectedTokens, ocrTokens)
		result.WER = werValue

		runesRef := []rune(expectedLower)
		runesOcr := []rune(ocrLower)
		if len(runesRef) > 0 {
			cerValue := float64(levenshtein.Distance(string(runesRef), string(runesOcr))) / float64(len(runesRef))
			result.CER = cerValue
		}
	}

	return result
}

// performEnhancedQualityChecks performs comprehensive image quality analysis
func (a *imageAnalyzer) performEnhancedQualityChecks(img image.Image, gray *image.Gray, result *AnalysisResult) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Resolution check
	result.Resolution = fmt.Sprintf("%dx%d", width, height)
	result.IsLowResolution = width < 750 || height < 1000

	// Brightness analysis
	brightness := a.calculateBrightness(gray)
	result.Brightness = brightness
	result.IsTooDark = brightness < 60
	result.IsTooBright = brightness > 200

	// Skew detection
	skewAngle := a.detectSkew(gray)
	if skewAngle != nil {
		result.SkewAngle = skewAngle
		result.IsSkewed = math.Abs(*skewAngle) > 5
	} else {
		result.IsSkewed = true // Unable to detect skew, assume skewed
	}

	// Edge detection and contour analysis
	numContours := a.detectContours(gray)
	result.NumContours = numContours
	result.HasDocumentEdges = numContours >= 1

	// OCR text coverage and confidence analysis
	a.analyzeOCRCoverage(gray, result)

	// QR code detection
	result.QRDetected = a.detectQRCode(img)
}

// calculateBrightness calculates the average brightness of a grayscale image
func (a *imageAnalyzer) calculateBrightness(gray *image.Gray) float64 {
	bounds := gray.Bounds()
	var sum float64
	pixelCount := float64(bounds.Dx() * bounds.Dy())

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			sum += float64(gray.GrayAt(x, y).Y)
		}
	}

	return sum / pixelCount
}

// detectSkew detects the skew angle of the document in the image
func (a *imageAnalyzer) detectSkew(gray *image.Gray) *float64 {
	bounds := gray.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Simple threshold to create binary image
	threshold := uint8(128)
	coords := make([][2]int, 0)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if gray.GrayAt(x, y).Y > threshold {
				coords = append(coords, [2]int{x, y})
			}
		}
	}

	if len(coords) == 0 {
		return nil
	}

	// Simple skew detection using line fitting
	// This is a simplified version - in practice you'd use more sophisticated methods
	angle := a.calculateSkewAngle(coords, width, height)

	// Normalize angle
	if angle < -45 {
		angle = -(90 + angle)
	} else {
		angle = -angle
	}

	// Handle near-vertical cases
	if math.Abs(angle-90) < 0.5 || math.Abs(angle-89.5) < 0.5 ||
		math.Abs(angle-90.5) < 0.5 || math.Abs(angle-89) < 0.5 ||
		math.Abs(angle-91) < 0.5 {
		angle = 0.0
	}

	return &angle
}

// calculateSkewAngle calculates skew angle from coordinates using simple linear regression
func (a *imageAnalyzer) calculateSkewAngle(coords [][2]int, width, height int) float64 {
	if len(coords) < 2 {
		return 0.0
	}

	// Sample a subset of coordinates for performance
	step := len(coords) / 1000
	if step < 1 {
		step = 1
	}

	var sumX, sumY, sumXY, sumX2 float64
	n := 0

	for i := 0; i < len(coords); i += step {
		x := float64(coords[i][0])
		y := float64(coords[i][1])
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
		n++
	}

	if n < 2 || sumX2 == 0 {
		return 0.0
	}

	// Linear regression to find slope
	slope := (float64(n)*sumXY - sumX*sumY) / (float64(n)*sumX2 - sumX*sumX)
	angle := math.Atan(slope) * 180 / math.Pi

	return angle
}

// detectContours detects contours in the image using simple edge detection
func (a *imageAnalyzer) detectContours(gray *image.Gray) int {
	bounds := gray.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Simple edge detection using Sobel-like operator
	edges := make([][]bool, height)
	for i := range edges {
		edges[i] = make([]bool, width)
	}

	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			// Simple gradient calculation
			gx := int(gray.GrayAt(x+1, y).Y) - int(gray.GrayAt(x-1, y).Y)
			gy := int(gray.GrayAt(x, y+1).Y) - int(gray.GrayAt(x, y-1).Y)
			magnitude := math.Sqrt(float64(gx*gx + gy*gy))

			edges[y][x] = magnitude > 50 // Threshold for edge detection
		}
	}

	// Count connected components (simplified contour counting)
	visited := make([][]bool, height)
	for i := range visited {
		visited[i] = make([]bool, width)
	}

	contourCount := 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if edges[y][x] && !visited[y][x] {
				a.floodFill(edges, visited, x, y, width, height)
				contourCount++
			}
		}
	}

	return contourCount
}

// floodFill performs flood fill algorithm for connected component labeling
func (a *imageAnalyzer) floodFill(edges, visited [][]bool, startX, startY, width, height int) {
	stack := [][2]int{{startX, startY}}

	for len(stack) > 0 {
		// Pop from stack
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		x, y := current[0], current[1]

		if x < 0 || x >= width || y < 0 || y >= height || visited[y][x] || !edges[y][x] {
			continue
		}

		visited[y][x] = true

		// Add neighbors to stack
		stack = append(stack, [2]int{x + 1, y}, [2]int{x - 1, y}, [2]int{x, y + 1}, [2]int{x, y - 1})
	}
}

// analyzeOCRCoverage analyzes OCR text coverage and confidence
func (a *imageAnalyzer) analyzeOCRCoverage(gray *image.Gray, result *AnalysisResult) {
	// Convert image to bytes for OCR processing
	buf := new(bytes.Buffer)
	err := png.Encode(buf, gray)
	if err != nil {
		result.TextCoverageRatio = 0
		result.LowTextCoverage = true
		result.AvgOCRConfidence = 0
		result.LowOCRConfidence = true
		return
	}

	// Set image for OCR
	a.tesseractClient.SetImageFromBytes(buf.Bytes())

	// Get OCR data with confidence scores
	// Note: gosseract doesn't provide detailed confidence data like pytesseract
	// This is a simplified implementation
	ocrText, err := a.tesseractClient.Text()
	if err != nil {
		result.TextCoverageRatio = 0
		result.LowTextCoverage = true
		result.AvgOCRConfidence = 0
		result.LowOCRConfidence = true
		return
	}

	// Simple text coverage estimation based on text length vs image size
	bounds := gray.Bounds()
	imageArea := float64(bounds.Dx() * bounds.Dy())
	textLength := float64(len(strings.TrimSpace(ocrText)))

	// Rough estimation: assume each character covers about 100 pixels
	estimatedTextArea := textLength * 100
	textCoverageRatio := estimatedTextArea / imageArea
	if textCoverageRatio > 1.0 {
		textCoverageRatio = 1.0
	}

	result.TextCoverageRatio = math.Round(textCoverageRatio*1000) / 1000 // Round to 3 decimal places
	result.LowTextCoverage = textCoverageRatio < 0.01

	// Simple confidence estimation based on text quality
	avgConfidence := a.estimateOCRConfidence(ocrText)
	result.AvgOCRConfidence = math.Round(avgConfidence*100) / 100 // Round to 2 decimal places
	result.LowOCRConfidence = avgConfidence < 40
}

// estimateOCRConfidence estimates OCR confidence based on text characteristics
func (a *imageAnalyzer) estimateOCRConfidence(text string) float64 {
	if len(text) == 0 {
		return 0
	}

	text = strings.TrimSpace(text)
	if len(text) == 0 {
		return 0
	}

	// Simple heuristic based on text characteristics
	confidence := 50.0 // Base confidence

	// Increase confidence for longer text
	if len(text) > 50 {
		confidence += 10
	}

	// Increase confidence for presence of common words
	words := strings.Fields(strings.ToLower(text))
	commonWords := []string{"the", "and", "or", "but", "in", "on", "at", "to", "for", "of", "with", "by"}
	commonWordCount := 0
	for _, word := range words {
		for _, common := range commonWords {
			if word == common {
				commonWordCount++
				break
			}
		}
	}

	if len(words) > 0 {
		commonWordRatio := float64(commonWordCount) / float64(len(words))
		confidence += commonWordRatio * 20
	}

	// Decrease confidence for too many special characters
	specialCharCount := 0
	for _, char := range text {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || char == ' ' || char == '.' ||
			char == ',' || char == '!' || char == '?') {
			specialCharCount++
		}
	}

	if len(text) > 0 {
		specialCharRatio := float64(specialCharCount) / float64(len(text))
		if specialCharRatio > 0.3 {
			confidence -= 20
		}
	}

	// Ensure confidence is within bounds
	if confidence > 100 {
		confidence = 100
	}
	if confidence < 0 {
		confidence = 0
	}

	return confidence
}

// detectQRCode detects QR codes in the image
func (a *imageAnalyzer) detectQRCode(img image.Image) bool {
	// Simple QR code detection using pattern matching
	// This is a simplified implementation since gozxing setup is complex
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Convert to grayscale for pattern detection
	gray := image.NewGray(bounds)
	draw.Draw(gray, bounds, img, bounds.Min, draw.Src)

	// Look for QR code finder patterns (simplified detection)
	// QR codes have distinctive square patterns in corners
	return a.hasQRPattern(gray, width, height)
}

// hasQRPattern performs simplified QR code pattern detection
func (a *imageAnalyzer) hasQRPattern(gray *image.Gray, width, height int) bool {
	// Look for square patterns that might indicate QR code finder patterns
	minSize := 7          // Minimum size for QR finder pattern
	maxSize := width / 10 // Maximum reasonable size

	if maxSize < minSize {
		return false
	}

	// Check corners for finder patterns
	corners := [][2]int{
		{0, 0},                // Top-left
		{width - maxSize, 0},  // Top-right
		{0, height - maxSize}, // Bottom-left
	}

	foundPatterns := 0
	for _, corner := range corners {
		if a.checkFinderPattern(gray, corner[0], corner[1], minSize, maxSize) {
			foundPatterns++
		}
	}

	// If we find at least 2 finder patterns, likely a QR code
	return foundPatterns >= 2
}

// checkFinderPattern checks for QR code finder pattern at given location
func (a *imageAnalyzer) checkFinderPattern(gray *image.Gray, startX, startY, minSize, maxSize int) bool {
	bounds := gray.Bounds()

	for size := minSize; size <= maxSize; size++ {
		if startX+size >= bounds.Max.X || startY+size >= bounds.Max.Y {
			continue
		}

		// Check for alternating dark-light-dark pattern
		centerX, centerY := startX+size/2, startY+size/2

		// Sample points in a cross pattern
		if a.isQRFinderPattern(gray, centerX, centerY, size/3) {
			return true
		}
	}

	return false
}

// isQRFinderPattern checks if the pattern at given center looks like QR finder pattern
func (a *imageAnalyzer) isQRFinderPattern(gray *image.Gray, centerX, centerY, radius int) bool {
	bounds := gray.Bounds()

	if centerX-radius < bounds.Min.X || centerX+radius >= bounds.Max.X ||
		centerY-radius < bounds.Min.Y || centerY+radius >= bounds.Max.Y {
		return false
	}

	// Check if center is dark
	centerPixel := gray.GrayAt(centerX, centerY).Y
	if centerPixel > 128 { // Not dark enough
		return false
	}

	// Check if corners are lighter (simplified check)
	corners := [][2]int{
		{centerX - radius, centerY - radius},
		{centerX + radius, centerY - radius},
		{centerX - radius, centerY + radius},
		{centerX + radius, centerY + radius},
	}

	lightCorners := 0
	for _, corner := range corners {
		if gray.GrayAt(corner[0], corner[1]).Y > 128 {
			lightCorners++
		}
	}

	// If most corners are light and center is dark, might be finder pattern
	return lightCorners >= 2
}
