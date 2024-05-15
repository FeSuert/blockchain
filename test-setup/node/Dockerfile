FROM golang:1.22

# Install system dependencies required for RocksDB
RUN apt-get update && apt-get install -y \
    build-essential \
    libgflags-dev \
    libsnappy-dev \
    zlib1g-dev \
    libbz2-dev \
    liblz4-dev \
    libzstd-dev \
    git \
    iputils-ping

# Install RocksDB from source
# RUN git clone https://github.com/facebook/rocksdb.git /tmp/rocksdb
# RUN cd /tmp/rocksdb && \
#     git checkout main && \
#     make shared_lib && \
#     make install && \
#     ldconfig

# Copy the Go project files into the container
COPY * /app/

# Set the working directory inside the container
WORKDIR /app/


# Download Go dependencies
RUN go mod download

# Build the Go application
RUN go build -o /app/node

# Cleanup to reduce the image size
RUN apt-get clean && \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# Specify the command to run on container start
CMD ["/app/node"]
