# This Dockerfile must stay buildable by the CLASSIC docker builder, not just
# BuildKit. testcontainers-go has no BuildKit support — integration/stack falls
# back to building this file through the daemon's classic build endpoint when no
# local checkout image exists, and BuildKit-only syntax fails there with
# "the --mount option requires BuildKit". So: no RUN --mount cache mounts, and
# no syntax directive. Layer ordering below does the caching work instead.

# Build stage.
#
# Alpine is used for the toolchain because go-sqlite3 requires cgo and we link
# statically against musl — see the build-static target in the Makefile.
ARG GO_VERSION=1.26.5
ARG ALPINE_VERSION=3.23

FROM public.ecr.aws/docker/library/golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS go_builder

# build-base provides gcc, musl-dev and binutils. make drives the build; git is
# needed because the Makefile shells out to it for version defaults.
RUN apk add --no-cache build-base git make

ENV CGO_ENABLED=1 GOOS=linux
WORKDIR /src

# Dependencies are copied and downloaded on their own so that editing source
# does not invalidate the module layer.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build metadata. Only the semver is passed in; the commit SHA, commit date and
# dirty flag are stamped by the toolchain from the copied .git (-buildvcs=true).
# Declared after the dependency layers so a new commit doesn't re-download modules.
ARG VERSION

RUN git config --global --add safe.directory /src && \
    make build-static VERSION="${VERSION}"

# Fail the build if the binary is not actually static. A dynamically linked
# binary would build fine here and then die with "no such file or directory" on
# distroless, which is a confusing way to find out.
RUN ! readelf -d /src/build/checkout | grep -q NEEDED

# Staged so the runtime image gets a /data owned by the nonroot user. Docker
# seeds a named volume from the image directory, ownership included, so without
# this the SQLite profile mounts a root-owned volume the service cannot write.
RUN mkdir -p /staging/data

############################################################################################################

# Runtime stage.
#
# distroless/static carries ca-certificates, tzdata and /etc/passwd, and nothing
# else — no shell, no package manager, no busybox. Container health checks go
# through `checkout health`, since there is no wget to call.
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=go_builder /src/build/checkout /usr/local/bin/checkout
COPY --from=go_builder --chown=nonroot:nonroot /staging/data /data
COPY LICENSE /LICENSE

USER nonroot:nonroot
WORKDIR /

# Expose the default application port
EXPOSE 8000

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=30 \
  CMD ["/usr/local/bin/checkout", "health"]

ENTRYPOINT ["/usr/local/bin/checkout"]
