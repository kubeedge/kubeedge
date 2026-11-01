# KubeEdge admission controller

This package runs the HTTPS admission webhook used by KubeEdge. It validates Device, DeviceModel, Rule, RuleEndpoint, and NodeUpgradeJob resources and mutates selected Pods and NodeUpgradeJobs.

## Local startup

Run these commands from the repository root.

First inspect the available flags:

```powershell
go run ./cloud/cmd/admission --help
```

Then start it with certificate paths and Service settings for your cluster:

```powershell
go run ./cloud/cmd/admission --kubeconfig "$env:USERPROFILE\.kube\config" --ca-cert-file "C:\path\to\ca.crt" --tls-cert-file "C:\path\to\server.crt" --tls-private-key-file "C:\path\to\server.key" --port 443 --webhook-namespace kubeedge --webhook-service-name kubeedge-admission-service
```

If the local process cannot bind to port 443, use an unprivileged port such as `9443` and configure the Kubernetes Service target port consistently.

## Verify

```powershell
gofmt -w cloud/pkg/admissioncontroller/*.go
go test ./cloud/pkg/admissioncontroller
```

Inspect the registered webhook configurations:

```powershell
kubectl get validatingwebhookconfiguration kubeedge-crds-validate-webhook-configuration -o yaml
kubectl get mutatingwebhookconfiguration kubeedge-mutating-webhook -o yaml
kubectl get mutatingwebhookconfiguration mutate-offlinemigration -o yaml
```

For an end-to-end check, create and update test Rules and RuleEndpoints, then create a Pod with the autonomy label both with and without existing `spec.tolerations`. The API server must reach the Service, and the CA bundle must trust the certificate served by this process.

## Runtime flow

`Run` builds Kubernetes clients, reads the CA bundle, registers webhooks, installs HTTP handlers, configures TLS, and starts the HTTPS server. Each handler receives an `AdmissionReview`, validates or mutates the embedded resource, and returns an `AdmissionResponse` containing the original request UID. The handler paths are `/devices`, `/devicemodels`, `/rules`, `/ruleendpoints`, `/nodeupgradejobs`, `/offlinemigration`, and `/mutating/nodeupgradejobs`.
