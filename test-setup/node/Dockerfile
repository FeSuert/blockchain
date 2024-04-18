FROM golang:1.21
COPY go.mod go.sum myApp.go /app/
WORKDIR /app/
RUN go mod download && \
    go build -o /app/node myApp.go
