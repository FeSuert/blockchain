FROM golang:1.21

# Install system dependencies required for RocksDB
RUN apt-get update && apt-get install -y \
    build-essential \
    libgflags-dev \
    libsnappy-dev \
    zlib1g-dev \
    libbz2-dev \
    liblz4-dev \
    libzstd-dev \
    git

# Install RocksDB from source
RUN git clone https://github.com/facebook/rocksdb.git /tmp/rocksdb && \
    cd /tmp/rocksdb && \
    git checkout v7.6.3 && \  # Check for the latest stable version to use
    make shared_lib && \
    make install && \
    ldconfig

# Copy the Go project files into the container
COPY go.mod go.sum myApp.go /app/

# Set the working directory inside the container
WORKDIR /app/

# Download Go dependencies
RUN go mod download

# Build the Go application
RUN go build -o /app/node myApp.go

# Cleanup to reduce the image size
RUN apt-get clean && \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# Specify the command to run on container start
CMD ["/app/node"]
