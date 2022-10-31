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
Return the proper image name
{{ include "common.images.image" ( dict "imageRoot" .Values.path.to.the.image "global" $) }}
*/}}
{{- define "common.images.image" -}}
{{- $registryName := .imageRoot.registry -}}
{{- $repositoryName := .imageRoot.repository -}}
{{- $separator := ":" -}}
{{- $termination := .imageRoot.tag | toString -}}
{{- if .global }}
    {{- if .global.imageRegistry }}
     {{- $registryName = .global.imageRegistry -}}
    {{- end -}}
{{- end -}}
{{- if .imageRoot.digest }}
    {{- $separator = "@" -}}
    {{- $termination = .imageRoot.digest | toString -}}
{{- end -}}
{{- printf "%s/%s%s%s" $registryName $repositoryName $separator $termination -}}
{{- end -}}

{{/*
Return the proper CloudCore image name
*/}}
{{- define "cloudCore.image" -}}
{{ include "common.images.image" (dict "imageRoot" .Values.cloudCore.image "global" .Values.global) }}
{{- end -}}

{{/*
Return the proper IptablesManager image name
*/}}
{{- define "iptablesManager.image" -}}
{{ include "common.images.image" (dict "imageRoot" .Values.iptablesManager.image "global" .Values.global) }}
{{- end -}}

{{/*
Return the proper controllerManager image name
*/}}
{{- define "controllerManager.image" -}}
{{ include "common.images.image" (dict "imageRoot" .Values.controllerManager.image "global" .Values.global) }}
{{- end -}}