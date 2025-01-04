{{/*
Expand the name of the chart.
*/}}
{{- define "vault-proxy-exporter.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "vault-proxy-exporter.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "vault-proxy-exporter.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "vault-proxy-exporter.labels" -}}
helm.sh/chart: {{ include "vault-proxy-exporter.chart" . }}
{{ include "vault-proxy-exporter.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "vault-proxy-exporter.selectorLabels" -}}
app.kubernetes.io/name: {{ include "vault-proxy-exporter.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "vault-proxy-exporter.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "vault-proxy-exporter.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Certificate secret name
*/}}
{{- define "vault-proxy-exporter.certifcateName" -}}
{{- if .Values.kubeRbacProxy.tls.certificateSecret }}
{{- .Values.kubeRbacProxy.tls.certificateSecret }}
{{- else }}
{{- include "vault-proxy-exporter.fullname" . }}-tls
{{- end }}
{{- end }}
