# Use Golang image based on Debian Bookworm
FROM golang:bookworm

# Set the working directory within the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go application
RUN go build -o main .

# Add environment variables for UID and GID
ARG DOCKER_UID=1000
ARG DOCKER_GID=1000

# Create a new group and user
RUN groupadd -g ${DOCKER_GID} appgroup && \
    useradd -u ${DOCKER_UID} -g appgroup -m appuser

# Change ownership of the working directory
RUN chown -R appuser:appgroup /app

# Switch to the new user
USER appuser

# Expose the port that the application will run on
EXPOSE 3335

# Set the command to run the executable
CMD ["./main"]
