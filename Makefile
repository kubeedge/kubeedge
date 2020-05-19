BINARIES=cloudcore \
	admission \
	edgecore \
	edgesite \
	keadm

COMPONENTS=cloud \
	edge

.EXPORT_ALL_VARIABLES:
OUT_DIR ?= _output

define BINARY_HELP_INFO
# Build binaries.
#
# Args:
#   WHAT: binary name to build. support: $(BINARIES)
#         the build will produce executable files under $(OUT_DIR)
#         If not specified, "everything" will be built.
#
# Example:
#   make
#   make all
#   make all HELP=y
#   make all WHAT=cloudcore
endef
.PHONY: all
ifeq ($(HELP),y)
all: clean
	@echo "$$BINARY_HELP_INFO"
else
all: verify-golang
	hack/make-rules/build.sh $(WHAT)
endif

define VERIFY_HELP_INFO
# verify golang,vendor and codegen
#
# Example:
# make verify
endef
.PHONY: verify
ifeq ($(HELP),y)
verify:
	@echo "$$VERIFY_HELP_INFO"
else
verify:verify-golang verify-vendor verify-codegen
endif

.PHONY: verify-golang
verify-golang:
	bash hack/verify-golang.sh

.PHONY: verify-vendor
verify-vendor:
	bash hack/verify-vendor.sh
.PHONY: verify-codegen
verify-codegen:
	bash cloud/hack/verify-codegen.sh

define TEST_HELP_INFO
# run golang test case.
#
# Args:
#   WHAT: Component names to be testd. support: $(COMPONENTS)
#         If not specified, "everything" will be tested.
#
# Example:
#   make test
#   make test HELP=y
#   make test WHAT=cloud
endef
.PHONY: test
ifeq ($(HELP),y)
test:
	@echo "$$TEST_HELP_INFO"
else
test: clean
	hack/make-rules/test.sh $(WHAT)
endif

define LINT_HELP_INFO
# run golang lint check.
#
# Example:
#   make lint
#   make lint HELP=y
endef
.PHONY: lint
ifeq ($(HELP),y)
lint:
	@echo "$$LINT_HELP_INFO"
else
lint:
	hack/make-rules/lint.sh
endif


INTEGRATION_TEST_COMPONENTS=edge
define INTEGRATION_TEST_HELP_INFO
# run integration test.
#
# Example:
#   make integrationtest
#   make integrationtest HELP=y
endef

.PHONY: integrationtest
ifeq ($(HELP),y)
integrationtest:
	@echo "$$INTEGRATION_TEST_HELP_INFO"
else
integrationtest:
	hack/make-rules/build.sh edgecore
	edge/test/integration/scripts/execute.sh
endif

ARCH ?= amd64
ARCH_VALUES=amd64 \
		 arm \
		 arm64
define CROSSBUILD_HELP_INFO
# Cross build components.
#
# Args:
#   WHAT: binary name to build. support: $(BINARIES)
#         the build will produce executable files under $(OUT_DIR)/$$ARCH
#         If not specified, "everything" will be built.
#
# ARCH: go arch value, now support: $(ARCH_VALUES)
#       If not specified ,default use ARCH=amd64
#
#
# Example:
#   make crossbuild
#   make crossbuild HELP=y
#   make crossbuild WHAT=edgecore
#   make crossbuild WHAT=edgecore ARCH=arm64
#
endef
.PHONY: crossbuild
ifeq ($(HELP),y)
crossbuild:
	@echo "$$CROSSBUILD_HELP_INFO"
else
crossbuild: clean
	ARCH=$(ARCH) hack/make-rules/crossbuild.sh $(WHAT)
endif

define E2E_HELP_INFO
# e2e test.
#
# Example:
#   make e2e
#   make e2e HELP=y
#
endef
.PHONY: e2e
ifeq ($(HELP),y)
e2e:
	@echo "$$E2E_HELP_INFO"
else
e2e:
#	bash tests/e2e/scripts/execute.sh device_crd
#	This has been commented temporarily since there is an issue of CI using same master for all PRs, which is causing failures when run parallelly
	bash tests/e2e/scripts/execute.sh
endif

define KEADM_E2E_HELP_INFO
# keadm e2e test.
#
# Example:
#   make keadm_e2e
#   make keadm_e2e HELP=y
#
endef
.PHONY: keadm_e2e
ifeq ($(HELP),y)
keadm_e2e:
	@echo "KEADM_E2E_HELP_INFO"
else
keadm_e2e:
	bash tests/e2e/scripts/keadm_e2e.sh
endif

define CLEAN_HELP_INFO
# Clean up the output of make.
#
# Example:
#   make clean
#   make clean HELP=y
#
endef
.PHONY: clean
ifeq ($(HELP),y)
clean:
	@echo "$$CLEAN_HELP_INFO"
else
clean:
	hack/make-rules/clean.sh
endif

VERSION ?= $(shell git describe --tags)

define RELEASE_HELP_INFO
# Release version.
#
# Example:
#   make release
#   make release HELP=y
#   make release VERSION=v1.3.0
#
endef
.PHONY: release
ifeq ($(HELP),y)
release:
	@echo "$$RELEASE_HELP_INFO"
else
release: clean
	hack/make-rules/release.sh $(VERSION)
endif

IMAGES=cloudcore \
	admission \
	edgecore \
	edgesite \
	csidriver \
	bluetooth

define IMAGE_HELP_INFO
# Build images.
#
# Args:
#   WHAT: component name to build. support: $(IMAGES)
#         If not specified, "everything" will be built.
#
# Example:
#   make image
#   make image HELP=y
#   make image WHAT=cloudcore
endef
.PHONY: image
ifeq ($(HELP),y)
image:
	@echo "$$IMAGE_HELP_INFO"
else
image:
	hack/make-rules/image.sh $(WHAT)
endif

.PHONY: bluetoothdevice
bluetoothdevice: clean
	hack/make-rules/bluetoothdevice.sh
