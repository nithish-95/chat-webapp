# Builder stage: compile the application with CGO enabled
FROM golang:alpine AS builder

# Install required packages for CGO (e.g., gcc and musl-dev)
RUN apk add --no-cache gcc musl-dev

# Enable CGO
ENV CGO_ENABLED=1

# Create working directory and binary output directory
RUN mkdir -p /app/bin
WORKDIR /app

# Copy all source code into the container
COPY . .

# Build the application; adjust main.go if your entry point is named differently
RUN go build -o /app/bin/chatapp .

# Final stage: build a minimal image
FROM alpine:latest

# Create a non-root user for security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Create and set the working directory
RUN mkdir /app
WORKDIR /app

# Copy the built binary and any HTML files from the builder stage
COPY --from=builder /app/bin/chatapp .
COPY --from=builder /app/*.html .

# Ensure the non-root user owns the files
RUN chown -R appuser:appgroup .

# Switch to the non-root user
USER appuser

# Expose the application port
EXPOSE 3000

# Start the application
ENTRYPOINT ["./chatapp"]



# FROM golang:alpine AS builder

# RUN mkdir -p /app/bin
# WORKDIR /app

# COPY . .

# RUN go build -o /app/bin/chatapp .

# FROM alpine:latest

# RUN addgroup -S appgroup
# RUN adduser -S appuser -G appgroup

# RUN mkdir /app
# WORKDIR /app

# COPY --from=builder /app/bin/chatapp .
# COPY --from=builder /app/*.html .

# RUN chown -R appuser:appgroup .
# USER appuser

# ENTRYPOINT ["./chatapp"]

# EXPOSE 3000