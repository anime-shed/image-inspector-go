package analyzer

import (
	"fmt"
	"math"

	"gocv.io/x/gocv"
)

type ImageAnalyzer interface {
	Analyze(imgPath string) (AnalysisResult, error)
}

type OpenCVAnalyzer struct{}

func NewOpenCVAnalyzer() ImageAnalyzer {
	return &OpenCVAnalyzer{}
}

func (a *OpenCVAnalyzer) Analyze(imgPath string) (AnalysisResult, error) {
	// Load image using OpenCV
	img := gocv.IMRead(imgPath, gocv.IMReadColor)
	if img.Empty() {
		return AnalysisResult{}, fmt.Errorf("could not read image: %s", imgPath)
	}
	defer img.Close()

	// Convert to HSV and LAB color spaces
	hsv := gocv.NewMat()
	defer hsv.Close()
	lab := gocv.NewMat()
	defer lab.Close()

	gocv.CvtColor(img, &hsv, gocv.ColorBGRToHSV)
	gocv.CvtColor(img, &lab, gocv.ColorBGRToLab)

	// Analyze different aspects
	result := AnalysisResult{
		Overexposed:   a.detectOverexposure(hsv),
		Oversaturated: a.detectOversaturation(hsv),
		IncorrectWB:   a.detectWhiteBalance(lab),
		Blurry:        a.detectBlur(img),
	}

	return result, nil
}

func (a *OpenCVAnalyzer) detectOverexposure(hsv gocv.Mat) bool {
	// Split HSV channels
	hsvChannels := gocv.Split(hsv)
	vChannel := hsvChannels[2]
	defer vChannel.Close()

	// Calculate histogram for value channel
	hist := gocv.NewMat()
	defer hist.Close()
	gocv.CalcHist([]gocv.Mat{vChannel}, []int{0}, gocv.NewMat(), &hist,
		[]int{256}, []float64{0, 256}, false)

	// Check percentage of pixels in highest 10% brightness range
	totalPixels := float64(vChannel.Rows() * vChannel.Cols())
	highBrightness := hist.GetFloatAt(230, 0) + hist.GetFloatAt(255, 0)
	return (highBrightness / totalPixels) > 0.25 // 25% pixels in bright range
}

func (a *OpenCVAnalyzer) detectOversaturation(hsv gocv.Mat) bool {
	// Split HSV channels
	hsvChannels := gocv.Split(hsv)
	sChannel := hsvChannels[1]
	defer sChannel.Close()

	// Calculate mean saturation
	mean := gocv.Mean(sChannel)
	return mean.Val1 > 150 // Saturation range 0-255
}

func (a *OpenCVAnalyzer) detectWhiteBalance(lab gocv.Mat) bool {
	// Split LAB channels
	labChannels := gocv.Split(lab)
	aChannel := labChannels[1]
	bChannel := labChannels[2]
	defer aChannel.Close()
	defer bChannel.Close()

	// Calculate mean values for A and B channels
	aMean := math.Abs(gocv.Mean(aChannel).Val1 - 128) // Neutral point is 128
	bMean := math.Abs(gocv.Mean(bChannel).Val1 - 128)

	return aMean > 15 || bMean > 15 // Allow ±15 deviation from neutral
}

func (a *OpenCVAnalyzer) detectBlur(img gocv.Mat) bool {
	// Convert to grayscale
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(img, &gray, gocv.ColorBGRToGray)

	// Calculate Laplacian variance
	laplacian := gocv.NewMat()
	defer laplacian.Close()
	gocv.Laplacian(gray, &laplacian, gocv.MatTypeCV64F, 1, 1, 0, gocv.BorderDefault)

	mean, stdDev := gocv.MeanStdDev(laplacian)
	variance := stdDev.Val1 * stdDev.Val1

	return variance < 100 // Lower variance indicates more blur
}

type AnalysisResult struct {
	Overexposed   bool `json:"overexposed"`
	Oversaturated bool `json:"oversaturated"`
	IncorrectWB   bool `json:"incorrect_white_balance"`
	Blurry        bool `json:"blurry"`
}
