# Build stage
FROM golang:1.23.5-bookworm AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    pkg-config cmake git build-essential libgtk2.0-dev \
    libavcodec-dev libavformat-dev libswscale-dev libv4l-dev \
    libxvidcore-dev libx264-dev libjpeg-dev libpng-dev libtiff-dev \
    && rm -rf /var/lib/apt/lists/*

# Clone OpenCV with explicit version
RUN git clone --depth 1 --branch 4.5.5 https://github.com/opencv/opencv.git /opencv && \
    git clone --depth 1 --branch 4.5.5 https://github.com/opencv/opencv_contrib.git /opencv_contrib

# Build OpenCV with ArUco support
RUN mkdir -p /opencv/build && \
    cd /opencv/build && \
    cmake \
    -D CMAKE_BUILD_TYPE=RELEASE \
    -D CMAKE_INSTALL_PREFIX=/usr/local \
    -D OPENCV_EXTRA_MODULES_PATH=/opencv_contrib/modules \
    -D BUILD_opencv_aruco=ON \
    -D OPENCV_GENERATE_PKGCONFIG=ON \
    -D BUILD_LIST=core,imgproc,aruco \
    .. && \
    make -j$(nproc) && \
    make install

# Configure Go environment
ENV PKG_CONFIG_PATH=/usr/local/lib/pkgconfig:$PKG_CONFIG_PATH
ENV CGO_CPPFLAGS="-I/usr/local/include/opencv4"
ENV CGO_LDFLAGS="-L/usr/local/lib -lopencv_core -lopencv_aruco -lopencv_imgproc"
RUN ldconfig

# Build application
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o /analyzer ./cmd/api/

# Runtime stage
FROM gcr.io/distroless/base-debian12:latest

# Copy OpenCV libraries and dependencies
COPY --from=builder /usr/local/lib/libopencv* /usr/local/lib/
COPY --from=builder /usr/lib/x86_64-linux-gnu/libgomp.so.1 /usr/lib/x86_64-linux-gnu/
COPY --from=builder /usr/lib/x86_64-linux-gnu/libstdc++.so.6 /usr/lib/x86_64-linux-gnu/
COPY --from=builder /analyzer /analyzer

# Environment variables
ENV LD_LIBRARY_PATH="/usr/local/lib:${LD_LIBRARY_PATH}"

# Non-root user setup
RUN addgroup --gid 65532 nonroot && \
    adduser --disabled-password --gecos "" --uid 65532 --ingroup nonroot nonroot

USER nonroot:nonroot

EXPOSE 8080
ENTRYPOINT ["/analyzer"]