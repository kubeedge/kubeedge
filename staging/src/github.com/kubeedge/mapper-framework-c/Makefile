SHELL := /bin/bash

curr_dir := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
parent_dir := $(abspath $(curr_dir)/..)

rest_args := $(wordlist 2, $(words $(MAKECMDGOALS)), $(MAKECMDGOALS))
$(eval $(rest_args):;@:)

mapper_name := $(or $(NAME),$(firstword $(rest_args)))
mapper_name := $(if $(mapper_name),$(mapper_name),mapper_default)

IMAGE ?= $(mapper_name)
TAG ?= latest

.PHONY: help generate build run clean_mapper

help:
	@echo "Usage:"
	@echo "  make generate [MapperName]  -> copy repo to sibling folder named MapperName"
	@echo "  make build [MapperName]     -> docker build (single Dockerfile)"
	@echo "  make run [MapperName]       -> docker run IMAGE:TAG"
	@echo "  make clean_mapper [Name]    -> rm -rf sibling folder"

generate:
	@chmod +x "$(curr_dir)/hack/make-rules/generate.sh"
	@"$(curr_dir)/hack/make-rules/generate.sh" "$(mapper_name)"

build:
	@chmod +x "$(curr_dir)/hack/make-rules/build.sh"
	@"$(curr_dir)/hack/make-rules/build.sh" "$(mapper_name)"

run:
	@echo "Running Docker image: $(IMAGE):$(TAG)"; \
    docker run --rm \
      -e MYSQL_ENABLED="$${MYSQL_ENABLED:-0}" \
      -e EDGECORE_SOCK="$${EDGECORE_SOCK:-/etc/kubeedge/dmi.sock}" \
      -e MAPPER_FORCE_FALLBACK="$${MAPPER_FORCE_FALLBACK:-1}" \
      -e MAPPER_MODBUS_ADDR="$${MAPPER_MODBUS_ADDR:-127.0.0.1}" \
      -e MAPPER_MODBUS_PORT="$${MAPPER_MODBUS_PORT:-1502}" \
      -v /etc/kubeedge/dmi.sock:/etc/kubeedge/dmi.sock \
      -v /tmp:/tmp \
      "$(IMAGE):$(TAG)"

clean_mapper:
	@echo "Removing: $(parent_dir)/$(mapper_name)"; \
    rm -rf -- "$(parent_dir)/$(mapper_name)"