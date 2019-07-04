# Feature Lifecycle

This document is to clarify definitions and differences between features and corresponding APIs
during different development stages (versions).

Each version has different level of stability, support time,
and requires different graduation criteria moving to next level:

* [Alpha](#alpha)
* [Beta](#beta)
* [GA](#ga)


## Alpha

The feature may be changed/upgraded in incompatible ways in the later versions.

The source code will be available in the release branch/tag as well as in the binaries.

Support for the feature can be stopped any time without notice.

The feature may have bugs.

The feature may also induce bugs in other APIs/Features if enabled.

The feature may not be completely implemented.

The API version names will be like v1alpha1, v1alpha2, etc. The suffixed number will be incremented by 1 in each upgrade.


### Graduation Criteria
-  Each feature will start at alpha level.
-  Should not break the functioning of other APIs/Features.


## Beta

The feature may not be changed/upgraded in incompatible ways in later versions,
but if changed in incompatible ways then upgrade strategy will be provided.

The source code will be available in the release branch/tag as well as in the binaries.

Support for the feature will not be stopped without 2 minor releases notice and will be present in at least next 2 minor releases.

The feature will have very less bugs.

The feature will not induce bugs in other APIs/Features if enabled.

The feature will be completely implemented.

The API version names will be like v1beta1, v1beta2, etc. The suffixed number will be incremented by 1 in each upgrade.

### Graduation Criteria
-  Should have at least 50% coverage in e2e tests.
-  Project agrees to support this feature for at least next 2 minor releases and notice of at least 2 minor releases will be given before stopping the support.
-  Feature Owner should commit to ensure backward/forward compatibility in the later versions.

## GA

The feature will not be changed/upgraded in incompatible ways in the next couple of versions.

The source code will be available in the release branch/tag as well as in the binaries.

Support for the feature will not be stopped without 4 minor releases notice and will be present in at least next 4 minor releases.

The feature will not have major bugs as it will be tested completely as well as have e2e tests.

The feature will not induce bugs in other APIs/Features if enabled.

The feature will be completely implemented.

The API version names will be like v1, v2, etc.


### Graduation Criteria
-  Should have complete e2e tests.
-  Code is thoroughly tested and is reported to be very stable.
-  Project will support this feature for at least next 4 minor releases and notice of at least 4 minor releases will be given before stopping support.
-  Feature Owner should commit to ensure backward/forward compatibility in the later versions.
-  Consensus from KubeEdge Maintainers as well as Feature/API Owners who use/interact with the Feature/API.
