# Stage 1: Build the Go binary using Go 1.24.2
FROM golang:1.24.2 AS builder

WORKDIR /app

# Copy go.mod and go.sum first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the app source code
COPY . .

# Build the binary
RUN go build -o /main .

# Stage 2: Run in a minimal base image
FROM debian:bookworm-slim

WORKDIR /app
COPY --from=builder /app/index.html ./index.html 
COPY --from=builder /app/message.css ./message.css 

# Copy the built binary from the builder stage
COPY --from=builder /main .

EXPOSE 8080

CMD ["./main"]