---
title: Dynamic Admission Control
status: implementable
authors:
    - "@chendave"
    - "@fisherxu"
approvers:
  - rohitsardesai83
  - fisherxu

creation-date: 2019-07-19
last-updated: 2019-09-11
---

# Motivation
As the evolving of the project, it is foreseeable more API and resource will be added in the project. kubeedge so far lack of effective way of pre-processing for the object configuration, for example, whether the pod that is going to be created contains the unwanted label, should the specific configmap be protected from deletion etc.

There is a concrete example on the github [issue 845](https://github.com/kubeedge/kubeedge/issues/845), the issue there cannot be addressed by [CRD validation](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#validation), so we need to explore other approach to achieve that purpose.


## Goals
* A framework to enable admission control in kubeedge.
* Address the edge cases where kubernetes CRD validation cannot accomplish.
* Create basic integration testcases.

## Non-goals
* Basic CRD validation should be done by [validation](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#validation).
* Start from `Device` and `DeviceModel` other kinds of resource will be evaluated later and will not be included in the first alpha version.

# Proposal
Propose using Kubernetes [Dynamic Admission Control](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers) to determine whether to accept or deny the request, the decision is based on how is the policy is configured, this gives us chance to validate the request before persisting the object.


# Design Details

## Admission service

Admission webhook is managed as an independent service, it could be built as a docker image and run as a standalone process, the feature could
be opted in by creating a k8s service with the built docker image as the backed image.

The entry point the service looks like this,

```golang
func main() {
	klog.InitFlags(nil)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	command := app.NewAdmissionCommand()

	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
```

A set of sample manifest is maintained in the code base, end-user could define their own manifest if there is any different usecases.

```bash
$ ls *.yaml
clusterrolebinding.yaml  clusterrole.yaml  deployment.yaml  serviceaccount.yaml  service.yaml
```


## Register admission webhook for each configuration
Rules could be centralized here, as an example, register one admission webhook as below,

```golang
Webhooks: []admissionregistrationv1beta1.ValidatingWebhook{
	{
		Name: ValidateDeviceModelWebhookName,
		Rules: []admissionregistrationv1beta1.RuleWithOperations{{
			Operations: []admissionregistrationv1beta1.OperationType{
				admissionregistrationv1beta1.Create,
				admissionregistrationv1beta1.Update,
			},
			Rule: admissionregistrationv1beta1.Rule{
				APIGroups:   []string{"devices.kubeedge.io"},
				APIVersions: []string{"v1alpha1"},
				Resources:   []string{"devicemodels"},
			},
		}},
		ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
			Service: &admissionregistrationv1beta1.ServiceReference{
				Namespace: opt.AdmissionServiceNamespace,
				Name:      opt.AdmissionServiceName,
				Path:      strPtr("/devicemodels"),
			},
			CABundle: cabundle,
		},
		FailurePolicy: &ignorePolicy,
	},
},
...
```

## Resource validation
The validation logic is registered as an http handler which responds to a specific HTTP request, each resource should have its own handler
pre-registered, and the validation is done by the handler.

```golang
http.HandleFunc("/devices", serveDevice)
func serveDevice(w http.ResponseWriter, r *http.Request) {
	serve(w, r, admitDevice)
}
```

```golang
func admitDevice(review admissionv1beta1.AdmissionReview) *admissionv1beta1.AdmissionResponse {
//validation logic is done here
}
```

## Certification management
Need to generate another set of key / certificate, or reuse the existing key / certificate for API validation, see [Authenticate apiservers](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#authenticate-apiservers) for details.
shell scripts will be provided to generate key / certificate.


## Basic integration test / unit testcases will be provided.
Testcase will cover all the basic usages and will be enriched with time goes on.

## New dependency
Below package is added as a new dependency.
* k8s.io/api/admission
