DESTDIR?=
USR_DIR?=/usr/local
INSTALL_DIR?=${DESTDIR}${USR_DIR}
INSTALL_BIN_DIR?=${INSTALL_DIR}/bin

# make all builds both cloud and edge binaries

BINARIES=cloudcore \
	admission \
	edgecore \
	edgesite-agent \
	edgesite-server \
	keadm \
	csidriver \
	iptablesmanager \
	edgemark \
	controllermanager \
	conformance

COMPONENTS=cloud \
	edge

.EXPORT_ALL_VARIABLES:
OUT_DIR ?= _output/local

BUILD_WITH_CONTAINER?=true
RUN = hack/make-rules/build_with_container.sh

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
#
#     By default we build inside container, only docker and make are required in this mode.
#     Set the environment variable or arg BUILD_WITH_CONTAINER to false to build with local environment. You need to install all dependencies to make building work.
#       1) make all WHAT=cloudcore BUILD_WITH_CONTAINER=false
#       2) export BUILD_WITH_CONTAINER=false && make all WHAT=cloudcore

endef
.PHONY: all
ifeq ($(HELP),y)
all: clean
	@echo "$$ALL_HELP_INFO"
else ifeq ($(BUILD_WITH_CONTAINER),true)
all:
	$(RUN) hack/make-rules/build.sh $(WHAT)
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
	hack/verify-codegen.sh
.PHONY: verify-vendor-licenses
verify-vendor-licenses:
	hack/verify-vendor-licenses.sh
.PHONY: verify-crds
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
	cloud/test/integration/scripts/execute.sh
endif

GOARM_VALUES=GOARM7 \
	GOARM8

define CROSSBUILD_HELP_INFO
# cross build components.
#
# Args:
#   WHAT: Component names to be lint check. support: $(BINARIES)
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
#   make crossbuild WHAT=edgecore BUILD_WITH_CONTAINER=false
#   make crossbuild WHAT=edgecore ARM_VERSION=GOARM7
#
endef
.PHONY: crossbuild
ifeq ($(HELP),y)
crossbuild:
	@echo "$$CROSSBUILD_HELP_INFO"
else ifeq ($(BUILD_WITH_CONTAINER),true)
crossbuild:
	$(RUN) hack/make-rules/crossbuild.sh $(WHAT) $(ARM_VERSION)
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

DMI_VERSION=v1alpha1
listDMIParams := DMI_VERSION
define GENERATE_DMI_HELP_INFO
# generate dmi api.pb.go from api.proto.
#
# Args:
#     DMI_VERSIONS, default: v1alpha1
#
# Example:
#     make dmi-proto
#     make dmi-proto -e DMI_VERSION=v1alpha1
#
endef
.PHONY: dmi-proto
ifeq ($(HELP),y)
dmi-proto:
	@echo "$$GENERATE_DMI_HELP_INFO"
else
dmi-proto:
	chmod a+x hack/generate-dmi-proto.sh && ./hack/generate-dmi-proto.sh $(foreach p, $(listDMIParams),--$(p)=$($(p)) )
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
#   make smallbuild WHAT=edgecore BUILD_WITH_CONTAINER=false
#
endef
.PHONY: smallbuild
ifeq ($(HELP),y)
smallbuild:
	@echo "$$SMALLBUILD_HELP_INFO"
else ifeq ($(BUILD_WITH_CONTAINER),true)
smallbuild:
	$(RUN) hack/make-rules/smallbuild.sh $(WHAT)
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
	tests/scripts/execute.sh
endif

define KEADM_DEPRECATED_E2E_HELP_INFO
# keadm e2e test.
#
# Example:
#   make keadm_deprecated_e2e
#   make keadm_deprecated_e2e HELP=y
#
endef
.PHONY: keadm_deprecated_e2e
ifeq ($(HELP),y)
keadm_deprecated_e2e:
	@echo "KEADM_DEPRECATED_E2E_HELP_INFO"
else
keadm_deprecated_e2e:
	$(RUN) hack/make-rules/release.sh kubeedge
	tests/scripts/keadm_deprecated_e2e.sh
endif

define KEADM_E2E_HELP_INFO
# eadm e2e test.
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
	tests/scripts/keadm_e2e.sh
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

define IMAGE_HELP_INFO
# Build image.
#
# Args:
#   WHAT: component names to build. support: $(BINARIES)
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
	@echo "IMAGE_HELP_INFO"
else
image:
	hack/make-rules/image.sh $(WHAT)
endif

define CROSS_IMAGE_HELP_INFO
# Use Buildx to build multi-architecture docker images.
#
# Args:
#   WHAT: component names to build. support: $(BINARIES)
#         If not specified, "everything" will be built.
#
# Example:
#   make crossbuildimage
#   make crossbuildimage HELP=y
#   make crossbuildimage WHAT=cloudcore
endef
.PHONY: crossbuildimage
ifeq ($(HELP),y)
crossbuildimage:
	@echo "CROSS_IMAGE_HELP_INFO"
else
crossbuildimage:
	hack/make-rules/crossbuildimage.sh $(WHAT)
endif

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
#   make release WHAT=kubeedge BUILD_WITH_CONTAINER=false
#   make release WHAT=kubeedge ARM_VERSION=GOARM7
#
endef
.PHONY: release
ifeq ($(HELP),y)
release:
	@echo "$$RELEASE_HELP_INFO"
else ifeq ($(BUILD_WITH_CONTAINER),true)
release:
	$(RUN) hack/make-rules/release.sh $(WHAT) $(ARM_VERSION)
else
release:
	hack/make-rules/release.sh $(WHAT) $(ARM_VERSION)
endif
