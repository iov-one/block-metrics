FROM golang:1.13.0-alpine AS build-env

ENV PACKAGES curl make git libc-dev bash gcc linux-headers eudev-dev python
ENV GO111MODULE=on

COPY . /block-metrics
WORKDIR /block-metrics

# Install minimum necessary dependencies, build Cosmos SDK, remove packages
RUN apk add --no-cache $PACKAGES && \
    make build

# Final image
FROM alpine:edge

WORKDIR /root

# Copy over binaries from the build-env
COPY --from=build-env /block-metrics/cmd/collector/collector /usr/bin/collector

CMD ["collector"]
