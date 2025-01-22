package analyzer

import (
	"image"
	"image/draw"
	"math"
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
}

type ImageAnalyzer interface {
	Analyze(img image.Image) AnalysisResult
}

type imageAnalyzer struct{}

func NewImageAnalyzer() ImageAnalyzer {
	return &imageAnalyzer{}
}

func (a *imageAnalyzer) Analyze(img image.Image) AnalysisResult {
	bounds := img.Bounds()
	gray := image.NewGray(bounds)
	draw.Draw(gray, bounds, img, bounds.Min, draw.Src)

	metrics := a.calculateMetrics(img, bounds)
	variance := a.computeLaplacianVariance(gray)

	return AnalysisResult{
		Overexposed:    metrics.avgLuminance > 200,
		Oversaturated:  metrics.avgSaturation > 0.6,
		IncorrectWB:    a.hasWhiteBalanceIssue(metrics.avgR, metrics.avgG, metrics.avgB),
		Blurry:         variance < 100,
		LaplacianVar:   variance,
		AvgLuminance:   metrics.avgLuminance,
		AvgSaturation:  metrics.avgSaturation,
		ChannelBalance: [3]float64{metrics.avgR, metrics.avgG, metrics.avgB},
	}
}

type metrics struct {
	avgLuminance, avgSaturation float64
	avgR, avgG, avgB            float64
}

func (a *imageAnalyzer) calculateMetrics(img image.Image, bounds image.Rectangle) metrics {
	var totalLum, totalSat, totalR, totalG, totalB float64
	pixelCount := float64(bounds.Dx() * bounds.Dy())

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			rf, gf, bf := float64(r>>8), float64(g>>8), float64(b>>8)

			_, s, v := a.rgbToHSV(rf, gf, bf)
			totalSat += s
			totalLum += v

			totalR += rf
			totalG += gf
			totalB += bf
		}
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
	maxAvg := math.Max(math.Max(avgR, avgG), avgB)
	minAvg := math.Min(math.Min(avgR, avgG), avgB)
	return maxAvg > 1.8*minAvg
}
