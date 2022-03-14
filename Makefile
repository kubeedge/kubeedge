DESTDIR?=
USR_DIR?=/usr/local
INSTALL_DIR?=${DESTDIR}${USR_DIR}
INSTALL_BIN_DIR?=${INSTALL_DIR}/bin
GOPATH?=$(shell go env GOPATH)

# make all builds both cloud and edge binaries

BINARIES=cloudcore \
	admission \
	edgecore \
	edgesite-agent \
	edgesite-server \
	keadm \
	csidriver

COMPONENTS=cloud \
	edge

.EXPORT_ALL_VARIABLES:
OUT_DIR ?= _output/local

define ALL_HELP_INFO
# Build code.
#
# Args:
#   WHAT: binary names to build. support: $(BINARIES)
#         the build will produce executable files under $(OUT_DIR)
#         If not specified, "everything" will be built.
#
# Example:
#   make
#   make all
#   make all HELP=y
#   make all WHAT=cloudcore
#   make all WHAT=cloudcore GOLDFLAGS="" GOGCFLAGS="-N -l"
#     Note: Specify GOLDFLAGS as an empty string for building unstripped binaries, specify GOGCFLAGS
#     to "-N -l" to disable optimizations and inlining, this will be helpful when you want to
#     use the debugging tools like delve. When GOLDFLAGS is unspecified, it defaults to "-s -w" which strips
#     debug information, see https://golang.org/cmd/link for other flags.

endef
.PHONY: all
ifeq ($(HELP),y)
all: clean
	@echo "$$ALL_HELP_INFO"
else
all: 
	KUBEEDGE_OUTPUT_SUBPATH=$(OUT_DIR) hack/make-rules/build.sh $(WHAT)
endif


define VERIFY_HELP_INFO
# verify golang, vendor, codegen and crds
#
# Example:
# make verify
endef
.PHONY: verify
ifeq ($(HELP),y)
verify:
	@echo "$$VERIFY_HELP_INFO"
else
verify:verify-golang verify-vendor verify-codegen verify-vendor-licenses verify-crds
endif

.PHONY: verify-golang
verify-golang:
	hack/verify-golang.sh

.PHONY: verify-vendor
verify-vendor:
	hack/verify-vendor.sh
.PHONY: verify-codegen
verify-codegen:
	cloud/hack/verify-codegen.sh
.PHONY: verify-vendor-licenses
verify-vendor-licenses:
	hack/verify-vendor-licenses.sh
.PHONY: verify-crds
verify-crds:
	hack/verify-crds.sh

define TEST_HELP_INFO
# run golang test case.
#
# Args:
#   WHAT: Component names to be testd. support: $(COMPONENTS)
#         If not specified, "everything" will be tested.
#   PROFILE: Generate profile named as "coverage.out"
#
# Example:
#   make test
#   make test HELP=y
#   make test PROFILE=y
#   make test WHAT=cloud
endef
.PHONY: test
ifeq ($(HELP),y)
test:
	@echo "$$TEST_HELP_INFO"
else ifeq ($(PROFILE),y)
test: clean
	PROFILE=coverage.out hack/make-rules/test.sh $(WHAT)
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
# Args:
#   WHAT: Component names to be lint check. support: $(INTEGRATION_TEST_COMPONENTS)
#         If not specified, "everything" will be integration check.
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

CROSSBUILD_COMPONENTS=edgecore
GOARM_VALUES=GOARM7 \
	GOARM8

define CROSSBUILD_HELP_INFO
# cross build components.
#
# Args:
#   WHAT: Component names to be lint check. support: $(CROSSBUILD_COMPONENTS)
#         If not specified, "everything" will be cross build.
#
# ARM_VERSION: go arm value, now support:$(GOARM_VALUES)
#        If not specified, build binary for ARMv8 by default.
#
#
# Example:
#   make crossbuild
#   make crossbuild HELP=y
#   make crossbuild WHAT=edgecore
#   make crossbuild WHAT=edgecore ARM_VERSION=GOARM7
#
endef
.PHONY: crossbuild
ifeq ($(HELP),y)
crossbuild:
	@echo "$$CROSSBUILD_HELP_INFO"
else
crossbuild: 
	hack/make-rules/crossbuild.sh $(WHAT) $(ARM_VERSION)
endif

CRD_VERSIONS=v1
CRD_OUTPUTS=build/crds
DEVICES_VERSION=v1alpha2
RELIABLESYNCS_VERSION=v1alpha1
listCRDParams := CRD_VERSIONS CRD_OUTPUTS DEVICES_VERSION RELIABLESYNCS_VERSION

define GENERATE_CRDS_HELP_INFO
# generate crds.
#
# Args:
#     CRD_VERSIONS, default: v1
#     CRD_OUTPUTS, default: build/crd
#     DEVICES_VERSION, default: v1alpha2
#     RELIABLESYNCS_VERSION, default: v1alpha1
#
# Example:
#     make generate 
#     make generate -e CRD_VERSIONS=v1 -e CRD_OUTPUTS=build/crds
#
endef
.PHONY: generate
ifeq ($(HELP),y)
generate:
	@echo "$$GENERATE_CRDS_HELP_INFO"
else
generate:
	chmod a+x hack/generate-crds.sh && ./hack/generate-crds.sh  $(foreach p, $(listCRDParams),--$(p)=$($(p)) )
endif

SMALLBUILD_COMPONENTS=edgecore
define SMALLBUILD_HELP_INFO
# small build components.
#
# Args:
#   WHAT: Component names to be lint check. support: $(SMALLBUILD_COMPONENTS)
#         If not specified, "everything" will be small build.
#
#
# Example:
#   make smallbuild
#   make smallbuild HELP=y
#   make smallbuild WHAT=edgecore
#
endef
.PHONY: smallbuild
ifeq ($(HELP),y)
smallbuild:
	@echo "$$SMALLBUILD_HELP_INFO"
else
smallbuild: 
	hack/make-rules/smallbuild.sh $(WHAT)
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
	tests/e2e/scripts/execute.sh
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
	tests/e2e/scripts/keadm_e2e.sh
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


QEMU_ARCH ?= x86_64
ARCH ?= amd64
IMAGE_TAG ?= $(shell git describe --tags)
GO_LDFLAGS='$(shell hack/make-rules/version.sh)'

.PHONY: cloudimage
cloudimage:
	docker build --build-arg GO_LDFLAGS=${GO_LDFLAGS} -t kubeedge/cloudcore:${IMAGE_TAG} -f build/cloud/Dockerfile .

.PHONY: admissionimage
admissionimage:
	docker build --build-arg GO_LDFLAGS=${GO_LDFLAGS} -t kubeedge/admission:${IMAGE_TAG} -f build/admission/Dockerfile .

.PHONY: csidriverimage
csidriverimage:
	docker build --build-arg GO_LDFLAGS=${GO_LDFLAGS} -t kubeedge/csidriver:${IMAGE_TAG} -f build/csidriver/Dockerfile .

.PHONY: iptablesmgrimage
iptablesmgrimage:
	docker build --build-arg GO_LDFLAGS=${GO_LDFLAGS} -t kubeedge/iptables-manager:${IMAGE_TAG} -f build/iptablesmanager/Dockerfile .

.PHONY: edgeimage
edgeimage:
	mkdir -p ./build/edge/tmp
	rm -rf ./build/edge/tmp/*
	curl -L -o ./build/edge/tmp/qemu-${QEMU_ARCH}-static.tar.gz https://github.com/multiarch/qemu-user-static/releases/download/v3.0.0/qemu-${QEMU_ARCH}-static.tar.gz
	tar -xzf ./build/edge/tmp/qemu-${QEMU_ARCH}-static.tar.gz -C ./build/edge/tmp
	docker build -t kubeedge/edgecore:${IMAGE_TAG} \
	--build-arg GO_LDFLAGS=${GO_LDFLAGS} \
	--build-arg BUILD_FROM=${ARCH}/golang:1.16-alpine3.13 \
	--build-arg RUN_FROM=${ARCH}/docker:dind \
	-f build/edge/Dockerfile .

.PHONY: edgesite-server-image
edgesite-server-image:
	docker build . --build-arg ARCH=${ARCH} -f build/edgesite/server-build.Dockerfile -t kubeedge/edgesite-server-${ARCH}:${IMAGE_TAG}

.PHONY: edgesite-agent-image
edgesite-agent-image:
	docker build . --build-arg ARCH=${ARCH} -f build/edgesite/agent-build.Dockerfile -t kubeedge/edgesite-agent-${ARCH}:${IMAGE_TAG}

define INSTALL_HELP_INFO
# install
#
# Args:
#   WHAT: Component names to be installed to $${INSTALL_BIN_DIR} (${INSTALL_BIN_DIR})
#         If not specified, "everything" will be installed
#
##
# Example:
#   make install
#   make install WHAT=edgecore
#
endef
.PHONY: help
ifeq ($(HELP),y)
install:
	@echo "$$INSTALL_HELP_INFO"
else
install: _output/local/bin
	install -d "${INSTALL_BIN_DIR}"
	if [ "" != "${WHAT}" ]; then \
          install "$</${WHAT}"  "${INSTALL_BIN_DIR}" ;\
        else \
          for file in ${BINARIES} ; do \
            install "$</$${file}"  "${INSTALL_BIN_DIR}" ;\
          done ; \
        fi
endif

define RELEASE_HELP_INFO
# release components.
#
# Args:
#   WHAT: Component names to be released. Support: kubeedge/edgesite/keadm
#         If not specified, "everything" will be built and released.
#
# Example:
#   make release
#   make release HELP=y
#   make release WHAT=kubeedge
#   make release WHAT=kubeedge ARM_VERSION=GOARM7
#
endef
.PHONY: release
ifeq ($(HELP),y)
release:
	@echo "$$RELEASE_HELP_INFO"
else
release:
	hack/make-rules/release.sh $(WHAT) $(ARM_VERSION)
endif
