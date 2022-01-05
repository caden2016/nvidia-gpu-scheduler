{{/*
Expand the name of the chart.
*/}}
{{- define "nvidia-gpu-scheduler.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "nvidia-gpu-scheduler.fullname" -}}
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
{{- define "nvidia-gpu-scheduler.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "nvidia-gpu-scheduler.labels" -}}
helm.sh/chart: {{ include "nvidia-gpu-scheduler.chart" . }}
{{ include "nvidia-gpu-scheduler.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "nvidia-gpu-scheduler.selectorLabels" -}}
app.kubernetes.io/name: {{ include "nvidia-gpu-scheduler.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "nvidia-gpu-scheduler.ds.selectorLabels" -}}
app.kubernetes.io/name: {{ include "nvidia-gpu-scheduler.name" . }}-ds
app.kubernetes.io/instance: {{ printf "%s-ds" .Release.Name  }}
{{- end }}

{{- define "nvidia-gpu-scheduler.apiservice.name" -}}
{{- printf "%s.%s" .Values.apiversion .Values.apigroup }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "nvidia-gpu-scheduler.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "nvidia-gpu-scheduler.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}
