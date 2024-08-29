# Use build arg hack to set the environment variable.
# NB! it will always remain what you passed during the build phase.
# ref: https://github.com/moby/moby/issues/37345#issuecomment-400245466
ARG PIPEDRIVE_API_SECRET

# Start from golang base image.
FROM golang:alpine AS builder

# Set the Pipedrive API key as a build arg.
ARG PIPEDRIVE_API_SECRET

# Add Maintainer info.
LABEL maintainer="Vladimir Rostok"

# Install git.
# Git is required for fetching the dependencies.
RUN apk update && apk add --no-cache git

# Set the current working directory inside the container.
WORKDIR /app

# Copy go mod and sum files.
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and the go.sum files are not changed.
RUN go mod download 

# Copy the source from the current directory to the working Directory inside the container.
COPY . .

# Set the Pipedrive API key as an environment variable.
ENV PIPEDRIVE_API_KEY=${PIPEDRIVE_API_SECRET}

# Build the Go app.
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main pipedrive-sync/cmd/sync.go

# Start a new stage from scratch.
FROM alpine:latest
RUN apk --no-cache add ca-certificates

# Set the Pipedrive API key as a build arg.
ARG PIPEDRIVE_API_SECRET

WORKDIR /app

# Create directory to place the configuration file.
RUN mkdir -p /app/sync/cmd/config

# Copy the Pre-built binary file from the previous stage and the configuration file.
COPY --from=builder /app/main .
COPY --from=builder /app/pipedrive-sync/cmd/config/configuration.yaml /app/config/

# Set the Pipedrive API key as an environment variable.
ENV PIPEDRIVE_API_KEY=${PIPEDRIVE_API_SECRET}

# Command to run the executable
CMD ["./main"]