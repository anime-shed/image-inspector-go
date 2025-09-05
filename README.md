# Go Image Analyzer

Go Image Analyzer is a web server application written in Go that fetches images from specified URLs, analyzes them for properties such as overexposure, oversaturation, incorrect white balance, and blurriness, and provides the results through an HTTP API. It also includes OCR quality validation to assess if images are suitable for OCR processing, though actual text extraction is not performed.

## Features

- Fetch images from specified URLs
- Analyze images for:
  - Overexposure
  - Oversaturation
  - Incorrect white balance
  - Blurriness
  - Laplacian variance (measure of sharpness)
  - Average luminance
  - Average saturation
  - Channel balance (red, green, blue)
- OCR quality validation:
  - Image quality assessment for OCR readiness
  - Word Error Rate (WER) calculation (when OCR is available)
  - Character Error Rate (CER) calculation (when OCR is available)
  - Detailed error reporting for image quality issues during OCR processing

## OCR Quality Validation

The OCR API now includes comprehensive quality validation that checks for the following conditions:

- **Resolution**: Minimum width of 800px, minimum height of 1000px, and minimum total area of 800,000 pixels
- **Blurriness**: Laplacian variance must be above 500.0 for acceptable sharpness
- **Brightness**: Must be between 80 and 220 (not too dark or too bright)
- **Overexposure/Oversaturation**: Checks for excessive light or color saturation
- **White Balance**: Ensures proper color balance across channels
- **Skew**: Document skew angle must be less than 5 degrees
- **Document Edges**: Verifies document edges are clearly visible
- **Contour Count**: Ensures sufficient contours for proper text recognition
- **Luminance and Saturation**: Validates average values are within acceptable ranges
- **Channel Balance**: Ensures RGB channels are properly balanced

If any of these conditions fail, the API response will include an "Errors" field with specific error messages.

## Prerequisites

- [Go](https://golang.org/doc/install) 1.16 or higher
- [Docker](https://docs.docker.com/get-docker/) (optional, for containerization)

## Installation

1. Clone the repository:
   ```sh
   git clone https://github.com/anime-shed/image-inspector-go.git
   cd image-inspector-go
   ```
2. Build the application:
   ```sh
   go build -o image-inspector-go ./cmd/api
   ```
```

## Usage

1. Set the necessary environment variables:
   ```sh
   export HOST=0.0.0.0
   export PORT=8080
   ```

2. Run the application:
   ```sh
   ./image-inspector-go
   ```

3. The server will start and listen on the specified address. You can interact with the API using tools like `curl` or Postman.

## Configuration

The application can be configured using environment variables. The following variables are available:

- `HOST`: The host address on which the server will listen (default: `0.0.0.0`).
- `PORT`: The port on which the server will listen (default: `8080`).
- `GIN_MODE`: The mode in which Gin should run (e.g., `release` for production).

## API Endpoints

### Basic Analysis
- `POST /analyze`: Analyze an image with optional OCR quality validation:
  - `url`: The URL of the image to be analyzed.
  - `is_ocr`: (optional) Boolean flag to enable OCR quality validation.
  - `expected_text`: (optional) When provided with `is_ocr=true`, triggers the OCR-comparison flow.

### Advanced Analysis Options
- `POST /analyze/options`: Analyze an image with custom analysis options:
  - `url`: The URL of the image to be analyzed.
  - `options`: Object with fields (all optional unless stated):
    - `ocr_mode` (boolean)
    - `fast_mode` (boolean)
    - `quality_mode` (boolean, default true)
    - `blur_threshold` (number)
    - `overexposure_threshold` (number)
    - `oversaturation_threshold` (number)
    - `luminance_threshold` (number)
    - `skip_qr_detection` (boolean)
    - `skip_white_balance` (boolean)
    - `skip_contour_detection` (boolean)
    - `skip_edge_detection` (boolean)
    - `use_worker_pool` (boolean, default true)
    - `max_workers` (integer, 0 = auto)

Example:
```json
{
  "url": "https://example.com/image.jpg",
  "options": {
    "quality_mode": true,
    "ocr_mode": false,
    "blur_threshold": 120.0,
    "skip_qr_detection": false,
    "use_worker_pool": true
  }
}
```

### Detailed Analysis (New)
- `POST /detailed-analyze`: Comprehensive image analysis with detailed metrics and thresholds:
  - `url`: The URL of the image to be analyzed.
  - `analysis_mode`: (optional) "basic" | "ocr" | "comprehensive" (default: "comprehensive")
  - `include_performance`: (optional) boolean
  - `include_raw_metrics`: (optional) boolean
  - `custom_thresholds`: (optional) object
  - `feature_flags`: (optional) object of booleans
  - `expected_text`: (optional) string (used only when `analysis_mode="ocr"`)
## Usage Examples

### Basic Image Analysis

```bash
curl -X POST http://localhost:8080/analyze \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com/image.jpg", "is_ocr": false}'
```

### OCR Quality Analysis

```bash
curl -X POST http://localhost:8080/analyze \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com/text-image.jpg", "is_ocr": true}'
```

### Detailed Analysis (Comprehensive)

```bash
curl -X POST http://localhost:8080/detailed-analyze \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com/image.jpg"}'
```

### Detailed Analysis with Custom Options

```bash
curl -X POST http://localhost:8080/detailed-analyze \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com/image.jpg",
    "analysis_mode": "comprehensive",
    "include_performance": true,
    "include_raw_metrics": true,
    "custom_thresholds": {
      "min_laplacian_variance": 100.0,
      "overexposure_threshold": 0.05,
      "oversaturation_threshold": 0.8,
      "min_total_pixels": 500000
    }
  }'
```

## Troubleshooting

### OCR Functionality

The OCR text extraction functionality has been removed from this version. The application will still perform image quality validation to determine if an image is suitable for OCR processing, but actual text extraction is not available.

If you need OCR text extraction functionality, please use a dedicated OCR service or library.

### Build Issues

If you encounter build errors:

1. **Check Go version:** Ensure you have Go 1.16 or higher installed
2. **Use Docker:** Build using the provided Dockerfile which includes all dependencies

### Sample Response (Basic OCR Quality Analysis)

```json
{
  "overexposed": false,
  "oversaturated": false,
  "incorrect_white_balance": false,
  "blurry": false,
  "laplacian_variance": 245.32,
  "average_luminance": 0.68,
  "average_saturation": 0.42,
  "channel_balance": [125.6, 130.2, 128.7],
  "word_error_rate": -1,
  "character_error_rate": -1,
  "ocr_error": "OCR processing is not available in this build",
  "errors": [
    "Image resolution is too low (600x800). Minimum requirements: 800x1000 or 800,000 total pixels",
    "Image is too blurry. Laplacian variance: 245.32 (minimum: 500.0)"
  ]
}
```

### Sample Response (Detailed Analysis)

```json
{
  "image_url": "https://example.com/image.jpg",
  "timestamp": "2024-01-15T10:30:00Z",
  "processing_time_sec": 1.42,
  "image_metadata": {
    "width": 1920,
    "height": 1080,
    "format": "image/jpeg",
    "content_type": "image/jpeg",
    "content_length": 245760
  },
  "quality_analysis": {
    "overexposed": false,
    "oversaturated": false,
    "incorrect_white_balance": false,
    "blurry": false,
    "is_low_resolution": false,
    "is_too_dark": false,
    "is_too_bright": false,
    "is_skewed": false,
    "is_valid": true,
    "is_ocr_ready": true,
    "has_critical_issues": false,
    "overall_quality_score": 85.5,
    "sharpness_score": 89.2,
    "exposure_score": 92.1,
    "color_score": 78.3
  },
  "raw_metrics": {
    "laplacian_variance": 150.0,
    "brightness": 128.0,
    "average_saturation": 0.5,
    "channel_balance": [128, 128, 128],
    "overexposed_pixel_ratio": 0.02,
    "underexposed_pixel_ratio": 0.05,
    "dynamic_range": 200,
    "total_pixels": 2073600,
    "aspect_ratio": 1.78
  },
  "applied_thresholds": {
    "min_laplacian_variance": 100,
    "overexposure_threshold": 0.1,
    "oversaturation_threshold": 0.8,
    "max_skew_angle": 5,
    "min_total_pixels": 10000
  },
  "quality_checks": [
    {
      "check_name": "blur_detection",
      "passed": true,
      "severity": "info",
      "actual_value": 150.0,
      "threshold_value": 100.0,
      "message": "Image sharpness is acceptable",
      "confidence": 0.85
    }
  ],
  "overall_assessment": {
    "quality_grade": "B",
    "usability_score": 85.5,
    "suitable_for": ["web", "display", "ocr"],
    "recommended_actions": []
  },
  "processing_details": {
    "analysis_mode": "comprehensive",
    "features_analyzed": ["sharpness", "exposure", "color", "resolution", "geometry"],
    "performance_metrics": {
      "total_processing_time_ms": 1420.67,
      "image_fetch_time_ms": 1200.45,
      "analysis_time_ms": 220.22
    }
  }
}
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Acknowledgments

This project makes use of several excellent open-source libraries and tools:

### Core Dependencies
- **[Gin Web Framework](https://github.com/gin-gonic/gin)** - High-performance HTTP web framework for Go
- **[Logrus](https://github.com/sirupsen/logrus)** - Structured logging for Go
- **[image](https://pkg.go.dev/image)** - Go's built-in image processing package

### Development Tools
- **[Docker](https://www.docker.com/)** - Containerization platform
- **[Alpine Linux](https://alpinelinux.org/)** - Security-oriented, lightweight Linux distribution used in Docker images

### Testing & Quality
- **[Testify](https://github.com/stretchr/testify)** - Testing toolkit for Go

We are grateful to the maintainers and contributors of these projects for their excellent work that makes this image analysis service possible.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
