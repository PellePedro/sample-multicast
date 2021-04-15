.PHONY: help setup build docker-build docker-run

BINARY   := halo
VERSION  := "0.1.0"
BUILD    := "1"
LDFLAGS  := -ldflags "-X main.Version=$(VERSION) -X main.Build=$(BUILD)"

CURRENT_DIR = $(shell pwd)

help:	## - Show help message
	@printf "\033[32m\xE2\x9c\x93 usage: make [target]\n\n\033[0m"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' Makefile | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'


build:	## - Building Application Container
	@printf "\033[32m\xE2\x9c\x93 Building Application Container ${REGISTRY}/${BINARY} \033[0m"
	@DOCKER_BUILDKIT=1 docker build \
		-t ${BINARY}:latest \
		--build-arg binary=$(BINARY) --build-arg build=$(BUILD) --build-arg version=$(VERSION) \
		-f Dockerfile .

run:	## - Running Application Container
	@printf "\033[32m\xE2\x9c\x93 Running Application Container ${REGISTRY}/${BINARY} \033[0m"
	@docker run -e CONTAINER_INTERFACE="eth0" ${BINARY}
