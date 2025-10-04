# Use a lightweight web server
FROM nginx:1.29.1-alpine

# Copy the built frontend assets
COPY . /usr/share/nginx/html

# Expose port 80
EXPOSE 80

# Start the server
CMD ["nginx", "-g", "daemon off;"]
