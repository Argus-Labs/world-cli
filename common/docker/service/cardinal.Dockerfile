################################
# Build Image - Normal
################################
FROM golang:1.24 AS build

ARG SOURCE_PATH
ARG GITHUB_TOKEN

WORKDIR /go/src/app

# Set Go environment variables for private repositories
ENV GOPRIVATE=github.com/argus-labs/*,pkg.world.dev/*

# Configure git to use HTTPS with GitHub token
RUN git config --global url."https://$(cat /run/secrets/github_token):x-oauth-basic@github.com/".insteadOf "https://github.com/"

# Copy go.mod files first to leverage Docker layer caching
COPY /${SOURCE_PATH}/go.mod ./

# Download dependencies
RUN go mod download

# Set the GOCACHE environment variable to /root/.cache/go-build to speed up build
ENV GOCACHE=/root/.cache/go-build

# Copy the entire source code
COPY /${SOURCE_PATH} ./

# Remove go.sum file
RUN rm go.sum

# Run go mod tidy to remove unused dependencies
RUN go mod tidy

# Build the binary
RUN go build -v -o /go/bin/app

################################
# Runtime Image - Normal
################################
FROM gcr.io/distroless/base-debian12 AS runtime

# Copy the binary from the build image
COPY --from=build /go/bin/app /usr/bin

# Run the binary
CMD ["app"]