# Use a  golang alpine as the base image
FROM public.ecr.aws/docker/library/golang:1.23.3-alpine3.20 as go_builder
RUN apk update
RUN apk add make cmake git alpine-sdk

# Setup

# Read arguments
ARG SERVICE
ARG VERSION_TAG
ARG GIT_COMMIT
ARG COMMIT_DATE
ARG BUILD_DATE

# Set env variables
ENV COMMIT_DATE=$COMMIT_DATE
ENV SERVICE=$SERVICE
ENV GIT_COMMIT=$GIT_COMMIT
ENV VERSION_TAG=$VERSION_TAG
ENV BUILD_DATE=$BUILD_DATE
RUN echo "building service: ${SERVICE}"
RUN echo "version: ${VERSION_TAG}"
RUN echo "git commit: ${GIT_COMMIT}"
RUN echo "commit date: ${COMMIT_DATE}"
RUN echo "compilation date: ${BUILD_DATE}"

# Set the working directory
WORKDIR /
COPY . .

# Build
RUN make gobuild

# Create linux svcuser
RUN mkdir /build/etc && \
    echo "svcuser:x:1010:1010::/sbin/nologin:/bin/false" > /build/etc/passwd && \
    echo "macuser:x:501:20::/sbin/nologin:/bin/false" >> /build/etc/passwd && \
    echo "linuxuser:x:1000:1000::/sbin/nologin:/bin/false" >> /build/etc/passwd && \
    echo "root:x:0:0:root:/sbin/nologin:/bin/false" >> /build/etc/passwd && \
    echo "svcgroup:x:1010:svcuser" > /build/etc/group && \
    echo "macgroup:x:20:macuser" >> /build/etc/group && \
    echo "linuxgroup:x:1000:linuxuser" >> /build/etc/group && \
    echo "root:x:0:root" >> /build/etc/group && \
    mkdir /build/config && \
    chown -R 1010:1010 /build/config


############################################################################################################

#SSL certs
FROM alpine as certs
RUN apk add --no-cache ca-certificates

############################################################################################################


# Copy binary to a scratch container. Let's keep our images nice and small!
FROM scratch
COPY --from=go_builder /build .
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY LICENSE ./
# Set User
USER svcuser
# Expose the port your application will run on
EXPOSE 8000

# Run the binary
ENTRYPOINT ["/checkout"]

