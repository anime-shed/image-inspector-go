package analyzer

import (
	"image"
	"math"
)

// metricsCalculator implements MetricsCalculator interface
type metricsCalculator struct{}

// NewMetricsCalculator creates a new metrics calculator
func NewMetricsCalculator() MetricsCalculator {
	return &metricsCalculator{}
}

// CalculateBasicMetrics computes basic image metrics like luminance and saturation
func (mc *metricsCalculator) CalculateBasicMetrics(img image.Image) metrics {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	totalPixels := float64(width * height)

	var totalLuminance, totalSaturation float64
	var totalR, totalG, totalB float64

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			// Convert from 16-bit to 8-bit
			rNorm := float64(r) / 65535.0
			gNorm := float64(g) / 65535.0
			bNorm := float64(b) / 65535.0

			totalR += rNorm
			totalG += gNorm
			totalB += bNorm

			// Calculate HSV for luminance and saturation
			_, s, v := mc.rgbToHSV(rNorm, gNorm, bNorm)
			totalLuminance += v
			totalSaturation += s
		}
	}

	return metrics{
		avgLuminance:  totalLuminance / totalPixels,
		avgSaturation: totalSaturation / totalPixels,
		avgR:          totalR / totalPixels,
		avgG:          totalG / totalPixels,
		avgB:          totalB / totalPixels,
	}
}

// CalculateLaplacianVariance computes the Laplacian variance for blur detection
func (mc *metricsCalculator) CalculateLaplacianVariance(gray *image.Gray) float64 {
	bounds := gray.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	var sum, sumSq float64
	count := 0

	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			// Laplacian kernel application
			center := float64(gray.GrayAt(x, y).Y)
			top := float64(gray.GrayAt(x, y-1).Y)
			bottom := float64(gray.GrayAt(x, y+1).Y)
			left := float64(gray.GrayAt(x-1, y).Y)
			right := float64(gray.GrayAt(x+1, y).Y)

			laplacian := -4*center + top + bottom + left + right
			sum += laplacian
			sumSq += laplacian * laplacian
			count++
		}
	}

	if count == 0 {
		return 0
	}

	mean := sum / float64(count)
	variance := (sumSq / float64(count)) - (mean * mean)
	return variance
}

// CalculateBrightness computes the average brightness of the image
func (mc *metricsCalculator) CalculateBrightness(gray *image.Gray) float64 {
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

// DetectSkew detects image skew angle using edge detection
func (mc *metricsCalculator) DetectSkew(gray *image.Gray) *float64 {
	bounds := gray.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Simple edge detection using Sobel operator
	var coords [][2]int
	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			// Sobel X kernel
			gx := -1*int(gray.GrayAt(x-1, y-1).Y) + 1*int(gray.GrayAt(x+1, y-1).Y) +
				-2*int(gray.GrayAt(x-1, y).Y) + 2*int(gray.GrayAt(x+1, y).Y) +
				-1*int(gray.GrayAt(x-1, y+1).Y) + 1*int(gray.GrayAt(x+1, y+1).Y)

			// Sobel Y kernel
			gy := -1*int(gray.GrayAt(x-1, y-1).Y) - 2*int(gray.GrayAt(x, y-1).Y) - 1*int(gray.GrayAt(x+1, y-1).Y) +
				1*int(gray.GrayAt(x-1, y+1).Y) + 2*int(gray.GrayAt(x, y+1).Y) + 1*int(gray.GrayAt(x+1, y+1).Y)

			magnitude := math.Sqrt(float64(gx*gx + gy*gy))
			if magnitude > 50 { // Threshold for edge detection
				coords = append(coords, [2]int{x, y})
			}
		}
	}

	if len(coords) < 10 {
		return nil
	}

	angle := mc.calculateSkewAngle(coords, width, height)
	return &angle
}

// calculateSkewAngle computes skew angle from edge coordinates
func (mc *metricsCalculator) calculateSkewAngle(coords [][2]int, width, height int) float64 {
	if len(coords) < 2 {
		return 0
	}

	// Simple linear regression to find dominant angle
	var sumX, sumY, sumXY, sumX2 float64
	n := float64(len(coords))

	for _, coord := range coords {
		x := float64(coord[0])
		y := float64(coord[1])
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	denominator := n*sumX2 - sumX*sumX
	if math.Abs(denominator) < 1e-10 {
		return 0
	}

	slope := (n*sumXY - sumX*sumY) / denominator
	angle := math.Atan(slope) * 180 / math.Pi

	// Normalize angle to [-45, 45] range
	for angle > 45 {
		angle -= 90
	}
	for angle < -45 {
		angle += 90
	}

	return angle
}

// DetectContours counts the number of contours in the image
func (mc *metricsCalculator) DetectContours(gray *image.Gray) int {
	bounds := gray.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Use a different approach: count distinct regions instead of edge components
	visited := make([][]bool, height)
	for i := range visited {
		visited[i] = make([]bool, width)
	}

	contourCount := 0
	threshold := 50 // Threshold for considering pixels as different regions

	// Find connected regions of similar pixels
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if !visited[y][x] {
				// Check if this region is significant enough to be a contour
				regionSize := mc.floodFillRegion(gray, visited, x, y, width, height, threshold)
				if regionSize > 25 { // Only count regions with at least 25 pixels
					contourCount++
				}
			}
		}
	}

	return contourCount
}

// floodFillRegion performs flood fill for similar pixel regions
func (mc *metricsCalculator) floodFillRegion(gray *image.Gray, visited [][]bool, startX, startY, width, height int, threshold int) int {
	if startX < 0 || startX >= width || startY < 0 || startY >= height || visited[startY][startX] {
		return 0
	}

	startValue := int(gray.GrayAt(startX, startY).Y)
	stack := [][2]int{{startX, startY}}
	regionSize := 0

	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		x, y := current[0], current[1]
		if x < 0 || x >= width || y < 0 || y >= height || visited[y][x] {
			continue
		}

		currentValue := int(gray.GrayAt(x, y).Y)
		if math.Abs(float64(currentValue-startValue)) > float64(threshold) {
			continue
		}

		visited[y][x] = true
		regionSize++
		stack = append(stack, [2]int{x + 1, y}, [2]int{x - 1, y}, [2]int{x, y + 1}, [2]int{x, y - 1})
	}

	return regionSize
}

// rgbToHSV converts RGB values to HSV color space
func (mc *metricsCalculator) rgbToHSV(r, g, b float64) (h, s, v float64) {
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