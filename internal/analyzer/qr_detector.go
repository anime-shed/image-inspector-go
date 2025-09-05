package analyzer

import (
	"image"
	"image/draw"
)

// qrDetector implements QRDetector interface
type qrDetector struct{}

// NewQRDetector creates a new QR detector
func NewQRDetector() QRDetector {
	return &qrDetector{}
}

// DetectQRCode detects if the image contains a QR code
func (qd *qrDetector) DetectQRCode(img image.Image) bool {
	// Convert to grayscale for processing
	bounds := img.Bounds()
	gray := image.NewGray(bounds)
	draw.Draw(gray, bounds, img, bounds.Min, draw.Src)

	width, height := bounds.Dx(), bounds.Dy()
	return qd.hasQRPattern(gray, width, height)
}

// hasQRPattern checks for QR code finder patterns in the image
func (qd *qrDetector) hasQRPattern(gray *image.Gray, width, height int) bool {
	// QR codes have finder patterns in corners
	// Look for 7x7 patterns with specific black/white ratios
	minSize := 7
	maxSize := min(width, height) / 3

	if maxSize < minSize {
		return false
	}

	// Check multiple positions for finder patterns
	positions := [][2]int{
		{width / 4, height / 4},     // Top-left area
		{3 * width / 4, height / 4}, // Top-right area
		{width / 4, 3 * height / 4}, // Bottom-left area
		{width / 2, height / 2},     // Center
	}

	patternCount := 0
	for _, pos := range positions {
		if qd.checkFinderPattern(gray, pos[0], pos[1], minSize, maxSize) {
			patternCount++
		}
	}

	// QR codes typically have at least 2-3 finder patterns
	return patternCount >= 2
}

// checkFinderPattern checks for QR finder pattern at a specific location
func (qd *qrDetector) checkFinderPattern(gray *image.Gray, startX, startY, minSize, maxSize int) bool {
	for size := minSize; size <= maxSize; size += 2 {
		radius := size / 2
		if qd.isQRFinderPattern(gray, startX, startY, radius) {
			return true
		}
	}
	return false
}

// isQRFinderPattern checks if the area around a point matches QR finder pattern
func (qd *qrDetector) isQRFinderPattern(gray *image.Gray, centerX, centerY, radius int) bool {
	bounds := gray.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Ensure we're within image bounds
	if centerX-radius < 0 || centerX+radius >= width ||
		centerY-radius < 0 || centerY+radius >= height {
		return false
	}

	// QR finder pattern has concentric squares: black-white-black-white-black
	// Sample points at different radii to check the pattern
	samples := []int{radius / 4, radius / 2, 3 * radius / 4, radius}
	expectedPattern := []bool{true, false, true, false} // black, white, black, white

	// Check pattern in multiple directions
	directions := [][2]int{{1, 0}, {0, 1}, {1, 1}, {-1, 1}}
	matchingDirections := 0

	for _, dir := range directions {
		matches := 0
		for i, sample := range samples {
			x := centerX + sample*dir[0]
			y := centerY + sample*dir[1]

			if x >= 0 && x < width && y >= 0 && y < height {
				pixelValue := gray.GrayAt(x, y).Y
				isBlack := pixelValue < 128

				if isBlack == expectedPattern[i] {
					matches++
				}
			}
		}

		// If most samples match the expected pattern
		if matches >= len(samples)-1 {
			matchingDirections++
		}
	}

	// Pattern should match in at least 2 directions
	return matchingDirections >= 2
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
