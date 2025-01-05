# Create a DockerFile for a golang 1.21 application
# Accept the Go version for the image to be set as a build argument.
ARG go_version=1.23

# Start from the latest golang base image
FROM golang:${go_version}-alpine AS builder

# Create the user and group files that will be used in the running container to
# run the process an unprivileged user.
RUN mkdir /user && \
    echo 'nobody:x:65534:65534:nobody:/:' > /user/passwd && \
    echo 'nobody:x:65534:' > /user/group

# Set the Current Working Directory inside the container
WORKDIR /app

RUN go env -w GOCACHE=/go-cache && \
    go env -w GOMODCACHE=/gomod-cache

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN --mount=type=cache,target=/gomod-cache go mod download

# Copy only specific source folders cmd, oauth2, pkg from the current directory to the Working Directory inside the container
COPY cmd cmd
COPY oauth2 oauth2
COPY pkg pkg
COPY main.go main.go

# Compile the golang application to a binary
RUN --mount=type=cache,target=/gomod-cache --mount=type=cache,target=/go-cache \
    go build -o dunebot main.go
RUN --mount=type=cache,target=/gomod-cache --mount=type=cache,target=/go-cache \
    go test -count 1 -failfast -timeout 3m ./...

# Create a application container from the scratch image and copy the binary to the container
FROM scratch

# Create a DockerFile for a golang 1.21 application
# Accept the Go version for the image to be set as a build argument.
ARG go_version

LABEL org.label-schema.vcs-url="https://github.com/containifyci/dunebot"
LABEL go-version="${go_version}"

# Import the user and group files.
COPY --from=builder /user/group /user/passwd /etc/
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Import the Certificate-Authority certificates for enabling HTTPS.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Import the compiled executable from the second stage.
COPY --from=builder --chown=nobody:nobody --chmod=770 /app/dunebot /app/dunebot

# Run the container as an unprivileged user.
USER nobody:nobody

ENTRYPOINT ["/app/dunebot"]
