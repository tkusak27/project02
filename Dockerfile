# Stage 1: Build the Go binary
FROM golang:1.23-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
RUN go build -o main .

# Stage 2: A minimal image to run the Go binary
FROM alpine:latest

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/main .

# Copy the templates and public directories
COPY --from=builder /app/templates ./templates
# Uncomment the next line if you have a public directory
COPY --from=builder /app/public ./public

# Copy the words.json file
COPY --from=builder /app/words.json .

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the Go binary
CMD ["./main"]
