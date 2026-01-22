{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "cloudcore.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "cloudcore.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Generate certificates for kubeedge cloudstream server
*/}}
{{- define "cloudcore.gen-certs" -}}
{{- $altNames := list ( printf "%s.%s" (include "cloudcore.name" .) .Release.Namespace ) ( printf "%s.%s.svc" (include "cloudcore.name" .) .Release.Namespace ) -}}
{{- $ca := genCA "cloudcore-ca" 365 -}}
{{- $cert := genSignedCert ( include "cloudcore.name" . ) nil $altNames 365 $ca -}}
streamCA.crt: {{ $ca.Cert | b64enc }}
stream.crt: {{ $cert.Cert | b64enc }}
stream.key: {{ $cert.Key | b64enc }}
{{- end -}}

{{/*
Return admission cert secret name
*/}}
{{- define "kubeedge.admission.certsSecretName" -}}
{{- if .Values.admission.certsSecretName -}}
{{ .Values.admission.certsSecretName }}
{{- else -}}
{{ printf "%s-admission-certs" .Release.Name }}
{{- end -}}
{{- end -}}

{{/*
Generate certificates for kubeedge admission
*/}}
{{- define "admission.gen-certs" -}}
{{- $altNames := list "kubeedge-admission-service" (printf "%s.%s" "kubeedge-admission-service" .Release.Namespace) (printf "%s.%s.svc" "kubeedge-admission-service" .Release.Namespace) -}}
{{- $ca := genCA (printf "%s.%s.svc" "kubeedge-admission-service" .Release.Namespace) 365 -}}
{{- $cert := genSignedCert (printf "%s.%s.svc" "kubeedge-admission-service" .Release.Namespace) nil $altNames 365 $ca -}}
ca.crt: {{ $ca.Cert | b64enc }}
tls.crt: {{ $cert.Cert | b64enc }}
tls.key: {{ $cert.Key | b64enc }}
{{- end -}}
