# Use a lightweight web server
FROM python:3.13.7-alpine3.22

# Copy the built frontend assets
COPY . .

# Expose port 80
EXPOSE 8080

USER 99

# Start the server
CMD ["python", "-m",  "http.server",  "8080"]
