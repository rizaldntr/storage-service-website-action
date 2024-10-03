FROM golang:1.23.2-alpine3.20 AS builder

# Turn on Go modules support and disable CGO
ENV GO111MODULE=on CGO_ENABLED=0

# Install upx (upx.github.io) to compress the compiled action
RUN apk --no-cache add upx binutils

# Set the working directory inside the container
WORKDIR /app

# Copy the Go modules and install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy all the files from the host into the container
COPY . .

# Compile the action - the added flags instruct Go to produce a
# standalone binary
RUN go build \
    -a \
    -trimpath \
    -ldflags "-s -w -extldflags '-static'" \
    -installsuffix cgo \
    -o ./bin/action \
    .

# Strip any symbols - this is not a library
RUN strip ./bin/action

# Compress the compiled action
RUN upx -q -9 ./bin/action

# Step 2

# Use the most basic and empty container - this container has no
# runtime, files, shell, libraries, etc.
FROM scratch

# Copy over SSL certificates from the first step - this is required
# if our code makes any outbound SSL connections because it contains
# the root CA bundle.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy over the compiled action from the first step
COPY --from=builder /app/bin/action /bin/action

# Specify the container's entrypoint as the action
ENTRYPOINT ["/bin/action"]