.PHONY: help generate-proto-stubs build-halo build-grpc-server run-halo run-local-halo

REGISTRY := docker.io/pellepedro
HALO_IMAGE := halo
GRPC_IMAGE := grpc-server
VERSION  := $(shell git describe --abbrev=0 --tags 2> /dev/null || echo "0.1.0")
BUILD    := $(shell git rev-parse --short HEAD 2> /dev/null || echo "undefined")
LDFLAGS  := -ldflags "-X main.Version=$(VERSION) -X main.Build=$(BUILD)"

CURRENT_DIR = $(shell pwd)

help:	## - Show help message
	@printf "\033[32m\xE2\x9c\x93 usage: make [target]\n\n\033[0m"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' Makefile | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

generate-proto-stubs:	## - Generate Protocol Buffer Stubs
	@printf "\033[32m\xE2\x9c\x93 Generate Protocol Buffer Stubs \033[0m"
	@docker build --target=artifact --output type=local,dest=${CURRENT_DIR}/protos/ -f Dockerfile.protoc  .

build-halo:	## - Building Halo Container
	@printf "\033[32m\xE2\x9c\x93 Building Application Container ${REGISTRY}/${BINARY} \033[0m"
	@docker build \
		-t ${HALO_IMAGE}:latest \
		-t ${HALO_IMAGE}:${VERSION} \
		--build-arg build=$(BUILD) --build-arg version=$(VERSION) \
		-f Dockerfile .

build-grpc-server:	## - Building GRPC Server
	@printf "\033[32m\xE2\x9c\x93 Buildingi Mock GRPC Server ${REGISTRY}/${BINARY} \033[0m"
	@DOCKER_BUILDKIT=1 docker build \
		-t ${GRPC_IMAGE}:latest \
		-t ${GRPC_IMAGE}:${VERSION} \
		--build-arg build=$(BUILD) --build-arg version=$(VERSION) \
		-f Dockerfile.grpc .

run-simulation: build-grpc-server build-halo ## - Run Simulation
	@printf "\033[32m\xE2\x9c\x93 Running Simulation \033[0m\n"
	@docker-compose up

purge-simulation: ## - Purge Simulation
	@printf "\033[32m\xE2\x9c\x93 Purge Simulation \033[0m\n"
	@docker-compose down
	@docker rm halo1 --force
	@docker rm halo2 --force
	@docker rm halo3 --force
	@docker rm grpc-server --force
	@sudo rm -rf /var/run/netns/halo*

