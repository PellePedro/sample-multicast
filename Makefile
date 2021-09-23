.PHONY: help build-halo run-halo

HALO_IMAGE := halo
VERSION  := $(shell git describe --abbrev=0 --tags 2> /dev/null || echo "0.1.0")
BUILD    := $(shell git rev-parse --short HEAD 2> /dev/null || echo "undefined")
LDFLAGS  := -ldflags "-X main.Version=$(VERSION) -X main.Build=$(BUILD)"

CURRENT_DIR = $(shell pwd)

help:	## - Show help message
	@printf "\033[32m\xE2\x9c\x93 usage: make [target]\n\n\033[0m"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' Makefile | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'


build-container:	## - Building Halo Container
	@printf "\033[32m\xE2\x9c\x93 Building Application Container ${REGISTRY}/${BINARY} \033[0m"
	@docker build \
		-t ${HALO_IMAGE}:latest \
		-t ${HALO_IMAGE}:${VERSION} \
		--build-arg build=$(BUILD) --build-arg version=$(VERSION) \
		-f Dockerfile .

build-halo: ## Build halo binary
	@rm -rf build && mkdir build
	@CGO_ENABLED=0 go build -gcflags="all=-N -l" \
	-ldflags="-X main.Version=${VERSION} -X main.Build=${BUILD} -X config.Build=${BUILD}" \
	-o build/halo cmd/tsf/halo/main.go

start-compose: build-container ## Start Docker Compose
	@docker compose up
