# Use a  golang alpine as the base image
FROM public.ecr.aws/docker/library/golang:1.25.1-alpine3.22 AS go_builder
RUN apk update
RUN apk add make cmake git alpine-sdk
ENV CGO_ENABLED=1 GOOS=linux
# Setup

# Read arguments
ARG SERVICE
ARG VERSION_TAG
ARG GIT_COMMIT
ARG COMMIT_DATE
ARG BUILD_DATE
ARG DIRTY
# Set env variables
ENV COMMIT_DATE=$COMMIT_DATE
ENV SERVICE=$SERVICE
ENV GIT_COMMIT=$GIT_COMMIT
ENV VERSION_TAG=$VERSION_TAG
ENV BUILD_DATE=$BUILD_DATE
ENV DIRTY=$DIRTY
RUN echo "building service: ${SERVICE}"
RUN echo "version: ${VERSION_TAG}"
RUN echo "git commit: ${GIT_COMMIT}"
RUN echo "commit date: ${COMMIT_DATE}"
RUN echo "compilation date: ${BUILD_DATE}"
RUN echo "dirty build: ${DIRTY}"

# Set the working directory
WORKDIR /
COPY . .

# Build
RUN make build

############################################################################################################

# Copy binary to a fresh alpine container. Let's keep our images nice and small!
FROM alpine:3.22
RUN adduser -D -H -u 1010 svcuser && apk add --no-cache ca-certificates
COPY --from=go_builder /build/checkout /checkout
COPY LICENSE ./
# Set User
USER svcuser
# Expose the default application port
EXPOSE 8080
# Run the binary
ENTRYPOINT ["/checkout"]

