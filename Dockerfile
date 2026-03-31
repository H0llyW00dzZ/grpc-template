# Copyright (c) 2026 H0llyW00dzZ All rights reserved.
#
# By accessing or using this software, you agree to be bound by the terms
# of the License Agreement, which you can find at LICENSE files.

FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the server
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server

# Final stage
FROM alpine:3.23

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /server .

# Expose gRPC port
EXPOSE 50051

# Run the server
CMD ["./server"]
