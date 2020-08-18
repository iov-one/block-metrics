.PHONY: install build dist test clean image

BUILD_VERSION ?= manual
BUILD_FLAGS := -mod=readonly
DOCKER_BUILD_FLAGS := -a -installsuffix cgo
BUILDOUT ?= collector
IMAGE_NAME = "iov1/block-metrics:${BUILD_VERSION}"

install:
	go install .

build:
	GOARCH=amd64 CGO_ENABLED=0 GOOS=linux go build $(BUILD_FLAGS) $(DOCKER_BUILD_FLAGS) -o $(BUILDOUT) .

dist: clean test build image

test:
	go vet -mod=readonly ./...
	go test -mod=readonly -race ./...

clean:
	rm -f ${BUILDOUT}

image:
	docker build --pull -t $(IMAGE_NAME) .
