# Use Go official image to build the binary
FROM golang:1.24.1 AS builder
WORKDIR /app

# Copy dependencies first
COPY go.mod go.sum ./
RUN go mod tidy && go mod download

# Copy the rest of the app
COPY . .

# Build the Go binary (statically linked)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main .

# Use Debian as the final image
FROM debian:bullseye-slim
WORKDIR /root/
COPY --from=builder /app/main .

# Expose the port and run the app
EXPOSE 8080
CMD ["./main"]
