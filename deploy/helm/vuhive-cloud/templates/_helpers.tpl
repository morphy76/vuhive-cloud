{{/*
Expand the name of the chart.
*/}}
{{- define "vuhive-cloud.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "vuhive-cloud.fullname" -}}
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
{{- define "vuhive-cloud.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "vuhive-cloud.labels" -}}
helm.sh/chart: {{ include "vuhive-cloud.chart" . }}
{{ include "vuhive-cloud.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "vuhive-cloud.selectorLabels" -}}
app.kubernetes.io/name: {{ include "vuhive-cloud.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "vuhive-cloud.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "vuhive-cloud.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Construct the PostgreSQL connection string.
*/}}
{{- define "vuhive-cloud.databaseUrl" -}}
{{- if .Values.database.url }}
{{- .Values.database.url }}
{{- else }}
{{- printf "postgres://%s:%s@%s:%d/%s?sslmode=%s" .Values.database.user .Values.database.password .Values.database.host (.Values.database.port | int) .Values.database.name .Values.database.sslmode }}
{{- end }}
{{- end }}

{{/*
Construct the runner API callback URL.
When runner.namespace differs from the release namespace, uses the cluster FQDN with a trailing dot
(http://<fullname>.<namespace>.svc.cluster.local.:<port>/api/v1/runs/complete) to prevent upstream ndots:5
search domain leaks. When runner and control plane share the same namespace, uses the unqualified
service name (http://<fullname>:<port>/api/v1/runs/complete).
*/}}
{{- define "vuhive-cloud.apiCallbackUrl" -}}
{{- if .Values.apiCallbackUrl }}
{{- .Values.apiCallbackUrl }}
{{- else if and .Values.runner.namespace (ne .Values.runner.namespace .Release.Namespace) }}
{{- printf "http://%s.%s.svc.cluster.local.:%d/api/v1/runs/complete" (include "vuhive-cloud.fullname" .) .Release.Namespace (.Values.service.port | int) }}
{{- else }}
{{- printf "http://%s:%d/api/v1/runs/complete" (include "vuhive-cloud.fullname" .) (.Values.service.port | int) }}
{{- end }}
{{- end }}

{{/*
Create a default fully qualified bff name.
*/}}
{{- define "vuhive-cloud.bff.fullname" -}}
{{- printf "%s-bff" (include "vuhive-cloud.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
BFF selector labels
*/}}
{{- define "vuhive-cloud.bff.selectorLabels" -}}
app.kubernetes.io/name: {{ include "vuhive-cloud.name" . }}-bff
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: bff
{{- end }}

{{/*
BFF common labels
*/}}
{{- define "vuhive-cloud.bff.labels" -}}
helm.sh/chart: {{ include "vuhive-cloud.chart" . }}
{{ include "vuhive-cloud.bff.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Construct the BFF upstream control plane URL.
*/}}
{{- define "vuhive-cloud.bff.controlPlaneUrl" -}}
{{- if (default (dict) .Values.bff).controlPlaneUrl }}
{{- .Values.bff.controlPlaneUrl }}
{{- else }}
{{- printf "http://%s:%d" (include "vuhive-cloud.fullname" .) (.Values.service.port | int) }}
{{- end }}
{{- end }}
