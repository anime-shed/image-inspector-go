# Go Image Analyzer

Go Image Analyzer is a web server application written in Go that fetches images from specified URLs, analyzes them for properties such as overexposure, oversaturation, incorrect white balance, and blurriness, and provides the results through an HTTP API.

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

## Usage

1. Set the necessary environment variables:
   ```sh
   export SERVER_ADDRESS=:8080
   ```

2. Run the application:
   ```sh
   ./image-inspector-go
   ```

3. The server will start and listen on the specified address. You can interact with the API using tools like `curl` or Postman.

## Configuration

The application can be configured using environment variables. The following variables are available:

- `SERVER_ADDRESS`: The address on which the server will listen (e.g., `:8080`).
- `GIN_MODE`: The mode in which Gin should run (e.g., `release` for production).

## API Endpoints

- `POST /inspect`: Analyze an image with additional options:
   - `url`: The URL of the image to be analyzed.
   - `is_ocr`: (optional) Boolean flag to enable OCR-specific thresholds.

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## License

This project is licensed under the MIT License - see the [MIT](MIT) file for details.
