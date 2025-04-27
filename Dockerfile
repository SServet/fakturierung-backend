FROM golang:1.23-alpine

# Install necessary tools
RUN apk add --no-cache git curl

# Install Air
RUN curl -sSfL https://raw.githubusercontent.com/cosmtrek/air/master/install.sh | sh -s -- -b /usr/local/bin

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum first to cache deps
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of your source code
COPY . .

# Expose app port
EXPOSE 8080

# Run Air for hot reload
CMD ["air"]
