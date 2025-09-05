package analyzer

import (
	"image"
	"math"
	"runtime"
	"sync"

	"gonum.org/v1/gonum/stat"
)

// metricsCalculator implements MetricsCalculator interface with Gonum optimizations
// Implements optimizations from PERFORMANCE_OPTIMIZATION_ANALYSIS.md Phase 2
type metricsCalculator struct {
	slicePool sync.Pool
}

// NewMetricsCalculator creates a new metrics calculator using Gonum
func NewMetricsCalculator() MetricsCalculator {
	return &metricsCalculator{
		slicePool: sync.Pool{
			New: func() interface{} {
				return make([]float64, 0, 1024)
			},
		},
	}
}

// CalculateBasicMetrics computes basic image metrics with parallel processing and Gonum optimizations
func (omc *metricsCalculator) CalculateBasicMetrics(img image.Image) metrics {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Handle empty images
	if width == 0 || height == 0 {
		return metrics{}
	}

	// Use parallel processing for better performance
	numWorkers := runtime.NumCPU()
	if height < numWorkers {
		numWorkers = height
		if numWorkers == 0 {
			numWorkers = 1
		}
	}
	rowsPerWorker := (height + numWorkers - 1) / numWorkers // ceil division

	type regionResult struct {
		lum, sat, r, g, b float64
		pixelCount        int
	}

	results := make(chan regionResult, numWorkers)
	var wg sync.WaitGroup

	// Process image in horizontal strips for better cache locality
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		startY := bounds.Min.Y + i*rowsPerWorker
		endY := startY + rowsPerWorker
		if i == numWorkers-1 || endY > bounds.Max.Y {
			endY = bounds.Max.Y
		}
		go func(startY, endY int) {
			defer wg.Done()

			var lum, sat, r, g, b float64
			pixelCount := 0

			for y := startY; y < endY && y < bounds.Max.Y; y++ {
				for x := bounds.Min.X; x < bounds.Max.X; x++ {
					rVal, gVal, bVal, _ := img.At(x, y).RGBA()
					// Convert from 16-bit to normalized float64
					rf := float64(rVal) / 65535.0
					gf := float64(gVal) / 65535.0
					bf := float64(bVal) / 65535.0

					_, s, v := omc.rgbToHSV(rf, gf, bf)
					sat += s
					lum += v
					r += rf
					g += gf
					b += bf
					pixelCount++
				}
			}

			results <- regionResult{lum, sat, r, g, b, pixelCount}
		}(startY, endY)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// Aggregate results using Gonum statistical functions
	var totalLum, totalSat, totalR, totalG, totalB float64
	totalPixelCount := 0

	for result := range results {
		totalLum += result.lum
		totalSat += result.sat
		totalR += result.r
		totalG += result.g
		totalB += result.b
		totalPixelCount += result.pixelCount
	}

	// Handle case where no pixels were processed
	if totalPixelCount == 0 {
		return metrics{}
	}

	pixelCount := float64(totalPixelCount)
	return metrics{
		avgLuminance:  totalLum / pixelCount,
		avgSaturation: totalSat / pixelCount,
		avgR:          totalR / pixelCount,
		avgG:          totalG / pixelCount,
		avgB:          totalB / pixelCount,
	}
}

// CalculateLaplacianVariance computes Laplacian variance using Gonum operations
func (omc *metricsCalculator) CalculateLaplacianVariance(gray *image.Gray) float64 {
	bounds := gray.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Get reusable slice from pool
	data := omc.slicePool.Get().([]float64)
	defer omc.slicePool.Put(data[:0])

	// Ensure capacity for all Laplacian values
	if cap(data) < (width-2)*(height-2) {
		data = make([]float64, 0, (width-2)*(height-2))
	}

	// Laplacian kernel: [0, 1, 0; 1, -4, 1; 0, 1, 0]
	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			center := float64(gray.GrayAt(x, y).Y)
			top := float64(gray.GrayAt(x, y-1).Y)
			bottom := float64(gray.GrayAt(x, y+1).Y)
			left := float64(gray.GrayAt(x-1, y).Y)
			right := float64(gray.GrayAt(x+1, y).Y)

			laplacian := -4*center + top + bottom + left + right
			data = append(data, laplacian)
		}
	}

	if len(data) == 0 {
		return 0
	}

	// Use Gonum's variance calculation
	return stat.Variance(data, nil)
}

// CalculateBrightness computes average brightness with parallel processing
func (omc *metricsCalculator) CalculateBrightness(gray *image.Gray) float64 {
	bounds := gray.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Handle empty images
	if width == 0 || height == 0 {
		return 0
	}

	// Use parallel processing for large images
	if width*height < 100000 {
		// For small images, use simple sequential processing
		return omc.calculateBrightnessSequential(gray)
	}

	numWorkers := runtime.NumCPU()
	if height < numWorkers {
		numWorkers = height
	}
	if numWorkers <= 0 {
		numWorkers = 1
	}
	rowsPerWorker := (height + numWorkers - 1) / numWorkers // ceil division

	results := make(chan float64, numWorkers)
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		startY := bounds.Min.Y + i*rowsPerWorker
		endY := startY + rowsPerWorker
		if i == numWorkers-1 || endY > bounds.Max.Y {
			endY = bounds.Max.Y
		}
		go func(startY, endY int) {
			defer wg.Done()

			var totalBrightness float64
			for y := startY; y < endY && y < bounds.Max.Y; y++ {
				for x := bounds.Min.X; x < bounds.Max.X; x++ {
					totalBrightness += float64(gray.GrayAt(x, y).Y)
				}
			}
			results <- totalBrightness
		}(startY, endY)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var totalBrightness float64
	for brightness := range results {
		totalBrightness += brightness
	}

	return totalBrightness / float64(width*height)
}

// calculateBrightnessSequential is a fallback for small images
func (omc *metricsCalculator) calculateBrightnessSequential(gray *image.Gray) float64 {
	bounds := gray.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	totalPixels := float64(width * height)

	var totalBrightness float64
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			totalBrightness += float64(gray.GrayAt(x, y).Y)
		}
	}

	return totalBrightness / totalPixels
}

// DetectSkew uses linear regression with Gonum
func (omc *metricsCalculator) DetectSkew(gray *image.Gray) *float64 {
	bounds := gray.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Simple edge detection using Sobel operator
	var xCoords, yCoords []float64
	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			// Sobel calculation
			gx := omc.calculateSobelX(gray, x, y)
			gy := omc.calculateSobelY(gray, x, y)

			magnitude := math.Sqrt(float64(gx*gx + gy*gy))
			if magnitude > 50 { // Threshold for edge detection
				xCoords = append(xCoords, float64(x))
				yCoords = append(yCoords, float64(y))
			}
		}
	}

	if len(xCoords) < 10 {
		return nil
	}

	// Use Gonum for linear regression
	angle := omc.calculateSkewAngle(xCoords, yCoords)
	return &angle
}

// calculateSobelX computes Sobel X gradient
func (omc *metricsCalculator) calculateSobelX(gray *image.Gray, x, y int) int {
	return -1*int(gray.GrayAt(x-1, y-1).Y) + 1*int(gray.GrayAt(x+1, y-1).Y) +
		-2*int(gray.GrayAt(x-1, y).Y) + 2*int(gray.GrayAt(x+1, y).Y) +
		-1*int(gray.GrayAt(x-1, y+1).Y) + 1*int(gray.GrayAt(x+1, y+1).Y)
}

// calculateSobelY computes Sobel Y gradient
func (omc *metricsCalculator) calculateSobelY(gray *image.Gray, x, y int) int {
	return -1*int(gray.GrayAt(x-1, y-1).Y) - 2*int(gray.GrayAt(x, y-1).Y) - 1*int(gray.GrayAt(x+1, y-1).Y) +
		1*int(gray.GrayAt(x-1, y+1).Y) + 2*int(gray.GrayAt(x, y+1).Y) + 1*int(gray.GrayAt(x+1, y+1).Y)
}

// calculateSkewAngle uses Gonum for linear regression
func (omc *metricsCalculator) calculateSkewAngle(xCoords, yCoords []float64) float64 {
	if len(xCoords) < 2 || len(yCoords) < 2 {
		return 0
	}

	// Use Gonum statistical functions for linear regression
	meanX := stat.Mean(xCoords, nil)
	meanY := stat.Mean(yCoords, nil)

	var sumXY, sumX2 float64
	for i := 0; i < len(xCoords); i++ {
		dx := xCoords[i] - meanX
		dy := yCoords[i] - meanY
		sumXY += dx * dy
		sumX2 += dx * dx
	}

	if math.Abs(sumX2) < 1e-10 {
		return 0
	}

	slope := sumXY / sumX2
	angle := math.Atan(slope) * 180 / math.Pi

	// Check for invalid angle values
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return 0
	}

	// Normalize angle to [-45, 45] range
	for angle > 45 {
		angle -= 90
	}
	for angle < -45 {
		angle += 90
	}

	return angle
}

// DetectContours performs basic contour detection using edge detection
func (omc *metricsCalculator) DetectContours(gray *image.Gray) int {
	bounds := gray.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Simple edge detection using Sobel operator
	edgeCount := 0
	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			// Sobel X
			gx := int(gray.GrayAt(x+1, y-1).Y) - int(gray.GrayAt(x-1, y-1).Y) +
				2*int(gray.GrayAt(x+1, y).Y) - 2*int(gray.GrayAt(x-1, y).Y) +
				int(gray.GrayAt(x+1, y+1).Y) - int(gray.GrayAt(x-1, y+1).Y)

			// Sobel Y
			gy := int(gray.GrayAt(x-1, y+1).Y) - int(gray.GrayAt(x-1, y-1).Y) +
				2*int(gray.GrayAt(x, y+1).Y) - 2*int(gray.GrayAt(x, y-1).Y) +
				int(gray.GrayAt(x+1, y+1).Y) - int(gray.GrayAt(x+1, y-1).Y)

			// Calculate magnitude
			magnitude := math.Sqrt(float64(gx*gx + gy*gy))
			if magnitude > 50 { // Threshold for edge detection
				edgeCount++
			}
		}
	}

	// Return approximate contour count (edges grouped)
	return edgeCount / 10 // Rough approximation
}

// rgbToHSV provides RGB to HSV conversion
func (omc *metricsCalculator) rgbToHSV(r, g, b float64) (h, s, v float64) {
	max := math.Max(r, math.Max(g, b))
	min := math.Min(r, math.Min(g, b))
	delta := max - min

	v = max

	if max == 0 {
		s = 0
	} else {
		s = delta / max
	}

	if delta == 0 {
		h = 0
	} else if max == r {
		h = 60 * (((g - b) / delta) + 0)
	} else if max == g {
		h = 60 * (((b - r) / delta) + 2)
	} else {
		h = 60 * (((r - g) / delta) + 4)
	}

	if h < 0 {
		h += 360
	}

	return h, s, v
}
