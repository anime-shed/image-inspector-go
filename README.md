# Go Image Analyzer

Go Image Analyzer is a web server application written in Go that fetches images from specified URLs, analyzes them for properties such as overexposure, oversaturation, incorrect white balance, and blurriness, and provides the results through an HTTP API. It also includes OCR (Optical Character Recognition) capabilities with error rate metrics calculation.

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
- OCR capabilities:
  - Text extraction from images
  - Word Error Rate (WER) calculation
  - Character Error Rate (CER) calculation
  - OCR confidence score

## Prerequisites

- [Go](https://golang.org/doc/install) 1.16 or higher
- [Tesseract OCR](https://github.com/tesseract-ocr/tesseract) 4.0 or higher (optional, required only for OCR functionality)
- [Docker](https://docs.docker.com/get-docker/) (optional, for containerization)

### Installing Tesseract OCR (Optional)

**Windows:**
```sh
# Using Chocolatey
choco install tesseract

# Or download from: https://github.com/UB-Mannheim/tesseract/wiki
```

**macOS:**
```sh
brew install tesseract
```

**Ubuntu/Debian:**
```sh
sudo apt-get install tesseract-ocr libtesseract-dev
```

## Installation

1. Clone the repository:
   ```sh
   git clone https://github.com/anime-shed/image-inspector-go.git
   cd image-inspector-go
   ```

2. Build the application:

   **Without OCR (basic image analysis only):**
   ```sh
   go build -o image-inspector-go ./cmd/api
   ```

   **With OCR (requires Tesseract OCR installed):**
   ```sh
   # On Windows
   set CGO_ENABLED=1 && go build -tags ocr -o image-inspector-go.exe ./cmd/api

   # On Linux/macOS
   CGO_ENABLED=1 go build -tags ocr -o image-inspector-go ./cmd/api
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

- `POST /analyze`: Analyze an image with optional OCR functionality:
   - `url`: The URL of the image to be analyzed.
   - `is_ocr`: (optional) Boolean flag to enable OCR processing.
   - `expected_text`: (optional) The expected text for error rate calculation when `is_ocr` is true.

## Usage Examples

### Basic Image Analysis

```bash
curl -X POST http://localhost:8080/analyze \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com/image.jpg", "is_ocr": false}'
```

### OCR Analysis with Error Metrics

```bash
curl -X POST http://localhost:8080/analyze \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com/text-image.jpg", "is_ocr": true, "expected_text": "This is the expected text in the image."}'
```

## Troubleshooting

### OCR Not Working

If OCR endpoints return an error about OCR not being available:

1. **Install Tesseract OCR** (see Prerequisites section)
2. **Rebuild with OCR support:**
   ```sh
   # Windows
   set CGO_ENABLED=1 && go build -tags ocr -o image-inspector-go.exe ./cmd/api

   # Linux/macOS
   CGO_ENABLED=1 go build -tags ocr -o image-inspector-go ./cmd/api
   ```

### Build Issues

If you encounter build errors related to CGO or Tesseract:

1. **Ensure CGO is enabled:** `set CGO_ENABLED=1` (Windows) or `export CGO_ENABLED=1` (Linux/macOS)
2. **Install development headers:** On Linux, install `libtesseract-dev` package
3. **Use Docker:** Build using the provided Dockerfile which includes all dependencies

### Sample Response (OCR Analysis)

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
  "ocr_text": "This is the detected text in the image.",
  "word_error_rate": 0.25,
  "character_error_rate": 0.15,
  "ocr_confidence": 92.5
}
```

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## License

This project is licensed under the MIT License - see the [MIT](MIT) file for details.
