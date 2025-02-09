# Stage 1: Build the application
FROM golang:bookworm AS builder

# Setup cache directories
RUN go env -w GOCACHE=/go-cache
RUN go env -w GOMODCACHE=/gomod-cache

WORKDIR /app

COPY . .

RUN --mount=type=cache,target=/gomod-cache --mount=type=cache,target=/go-cache \
    go build -ldflags="-w -s" -o haven .

# Final stage: Run the application
FROM gcr.io/distroless/base-debian12:nonroot

# Copy the built application
COPY --from=builder /app/haven .

# Expose the port that the application will run on
ARG RELAY_PORT=3355
EXPOSE ${RELAY_PORT}

# Set the command to run the executable
CMD ["./haven"]
