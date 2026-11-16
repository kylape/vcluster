# Build program
# Use the Red Hat go-toolset builder so the build does not depend on Docker Hub
# anonymous pull limits; it ships the Go version required by go.mod plus curl,
# git, and tar. Run as root because the image defaults to an unprivileged UID
# and the build writes to /usr/local/bin and the root cache directories.
FROM registry.access.redhat.com/ubi9/go-toolset:1.26.10 AS builder
USER root

WORKDIR /vcluster-dev
ARG TARGETOS
ARG TARGETARCH
ARG BUILD_VERSION=dev
ARG TELEMETRY_PRIVATE_KEY=""
ARG HELM_VERSION="v3.22.0"

# Install kubectl for development
RUN curl -LO "https://dl.k8s.io/release/$(curl -fsSL https://dl.k8s.io/release/stable.txt)/bin/linux/${TARGETARCH}/kubectl" && chmod +x ./kubectl && mv ./kubectl /usr/local/bin/kubectl

# Install helm binary
RUN curl -s https://get.helm.sh/helm-${HELM_VERSION}-linux-${TARGETARCH}.tar.gz > helm3.tar.gz && tar -zxvf helm3.tar.gz linux-${TARGETARCH}/helm && chmod +x linux-${TARGETARCH}/helm && mv linux-${TARGETARCH}/helm /usr/local/bin/helm && rm helm3.tar.gz && rm -R linux-${TARGETARCH}

# Install Delve for debugging
RUN if [ "${TARGETARCH}" = "amd64" ] || [ "${TARGETARCH}" = "arm64" ]; then go install github.com/go-delve/delve/cmd/dlv@latest; fi

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
COPY vendor/ vendor/

# Copy the go source
COPY cmd/vcluster cmd/vcluster
COPY cmd/vclusterctl cmd/vclusterctl
COPY pkg/ pkg/
COPY config/ config/

ENV GO111MODULE=on
ENV DEBUG=true
# go-toolset defaults GOPATH to /opt/app-root/src/go; pin it to /go so the
# build cache mounts below keep targeting the module and build caches.
ENV GOPATH=/go

# create and set GOCACHE now, this should slightly speed up the first build inside of the container
# also create /.config folder for GOENV, as dlv needs to write there when starting debugging
RUN mkdir -p /.cache /.config
ENV GOCACHE=/.cache
ENV GOENV=/.config

# Set home to "/" in order to for kubectl to automatically pick up vcluster kube config
ENV HOME=/

# Build cmd
# Constrain Go build parallelism: the CI build environment limits threads
# and processes per container, and an unconstrained build dies with
# "fatal error: newosproc" (errno=11) when the compiler spawns workers.
RUN --mount=type=cache,id=gomod,target=/go/pkg/mod \
	--mount=type=cache,id=gobuild,target=/.cache/go-build \
	GOMAXPROCS=4 CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GO111MODULE=on go build -p 4 -mod vendor -ldflags "-X github.com/loft-sh/vcluster/pkg/telemetry.SyncerVersion=$BUILD_VERSION -X github.com/loft-sh/vcluster/pkg/telemetry.telemetryPrivateKey=$TELEMETRY_PRIVATE_KEY" -o /vcluster cmd/vcluster/main.go

# RUN useradd -u 12345 nonroot
# USER nonroot

ENTRYPOINT ["go", "run", "-mod", "vendor", "cmd/vcluster/main.go", "start"]

# Use Red Hat UBI for the runtime image so the published image has a Red Hat
# supported base and can run under OpenShift security policies.
FROM registry.access.redhat.com/ubi9-minimal:latest

# install runtime dependencies
RUN microdnf install -y ca-certificates zstd tzdata \
    && microdnf clean all \
    && rm -rf /var/cache/yum

# Set root path as working directory
WORKDIR /

COPY --from=builder /vcluster .

# OpenShift may assign an arbitrary non-root UID. Runtime code must use a
# writable home directory rather than assuming /root is available.
ENV HOME=/tmp

# RUN useradd -u 12345 nonroot
# USER nonroot

ENTRYPOINT ["/vcluster", "start"]
