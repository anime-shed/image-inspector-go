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

- `POST /analyze`: Analyze an image with optional OCR quality validation:
   - `url`: The URL of the image to be analyzed.
   - `is_ocr`: (optional) Boolean flag to enable OCR quality validation.
   - `expected_text`: (optional) This parameter is retained for API compatibility but is not used in the current version.

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

## Troubleshooting

### OCR Functionality

The OCR text extraction functionality has been removed from this version. The application will still perform image quality validation to determine if an image is suitable for OCR processing, but actual text extraction is not available.

If you need OCR text extraction functionality, please use a dedicated OCR service or library.

### Build Issues

If you encounter build errors:

1. **Check Go version:** Ensure you have Go 1.16 or higher installed
2. **Use Docker:** Build using the provided Dockerfile which includes all dependencies

### Sample Response (OCR Quality Analysis)

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

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## License

This project is licensed under the MIT License - see the [MIT](MIT) file for details.
