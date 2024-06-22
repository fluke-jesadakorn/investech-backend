# Use the official Golang image as a parent image
FROM golang:1.22.4-alpine3.20

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Download necessary Go modules
RUN go mod download

# Build the Go app
RUN go build -o main .

# Expose port (ensure it matches the one your application is set to use)
EXPOSE 8080

# Command to run the executable
CMD ["./main"]
