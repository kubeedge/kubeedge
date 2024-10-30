SHELL := /bin/bash
curr_dir := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
rest_args := $(wordlist 2, $(words $(MAKECMDGOALS)), $(MAKECMDGOALS))
$(eval $(rest_args):;@:)

help:
	@echo "Usage:"
	@echo "  make generate [MapperName] : generate a mapper based on the Mapper-framework."
	@echo "                               If MapperName is not provided, the default value 'mapper_default' is used."
	@echo "  make build [ImageName]     : build Docker images."
	@echo "                               If ImageName is not provided, the name of the current project is used as the image name."
	@echo

make_rules := $(shell ls $(curr_dir)/hack/make-rules | sed 's/.sh//g')
$(make_rules):
	@chmod +x $(curr_dir)/hack/make-rules/$@.sh
	@$(curr_dir)/hack/make-rules/$@.sh $(rest_args)