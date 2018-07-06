FROM golang:1.10 AS builder

# Download and install the latest release of dep
ADD https://github.com/golang/dep/releases/download/v0.4.1/dep-linux-amd64 /usr/bin/dep
RUN chmod +x /usr/bin/dep

# Copy the code from the host and compile it
WORKDIR $GOPATH/src/github.com/bege13mot/consignment-service
COPY Gopkg.toml Gopkg.lock ./
RUN dep ensure --vendor-only
COPY . ./

# CGO_ENABLED=0 is an environment variable that tells to the compiler to disable the support for linking C code. As a result, the resulting binary will not be able to depend on the C system libraries. The point is that in a scratch container, there are no system libraries. If we omit this directive, the Docker build will terminate successfully, but the resulting container will crash with funny errors.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-X main.Version=$(git rev-parse HEAD) -X main.Branch=$(git rev-parse --abbrev-ref HEAD) -X main.BuildTime=$(date -u '+%Y-%m-%d_%H:%M:%S')" -a -installsuffix nocgo -o /consignment-service .

FROM scratch
COPY --from=builder /consignment-service ./
ENTRYPOINT ["./consignment-service"]
