################################
# Build Image - Normal
################################
FROM golang:1.24 AS build

ARG SOURCE_PATH
ARG GITHUB_TOKEN

WORKDIR /go/src/app

# Set Go environment variables for private repositories
ENV GOPRIVATE=github.com/argus-labs/*,pkg.world.dev/*

# Configure git to use HTTPS with GitHub token if provided, otherwise use SSH
RUN git config --global url."https://${GITHUB_TOKEN}:x-oauth-basic@github.com/".insteadOf "https://github.com/"

# Copy the entire source code
COPY /${SOURCE_PATH} ./

# Download dependencies
RUN go mod tidy

# Set the GOCACHE environment variable to /root/.cache/go-build to speed up build
ENV GOCACHE=/root/.cache/go-build

# Build the binary
RUN --mount=type=cache,target="/root/.cache/go-build" go build -v -o /go/bin/app

################################
# Runtime Image - Normal
################################
FROM gcr.io/distroless/base-debian12 AS runtime

# Copy the binary from the build image
COPY --from=build /go/bin/app /usr/bin

# Run the binary
CMD ["app"]