# Use a minimal base image
FROM alpine:latest

# Set a working directory
WORKDIR /app

# Default command does nothing
CMD ["sleep", "infinity"]