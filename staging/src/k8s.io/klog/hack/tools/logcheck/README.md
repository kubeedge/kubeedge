This directory contains tool for checking use of unstructured logs in a package. It is created to prevent regression after packages have been migrated to use structured logs.

**Installation:**`go install k8s.io/klog/hack/tools/logcheck`
**Usage:** `$logcheck.go <package_name>`
`e.g $logcheck ./pkg/kubelet/lifecycle/`
