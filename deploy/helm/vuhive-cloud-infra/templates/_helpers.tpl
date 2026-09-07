{{/*
Expand the name of the chart.
*/}}
{{- define "vuhive-cloud-infra.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "vuhive-cloud-infra.fullname" -}}
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
Common labels
*/}}
{{- define "vuhive-cloud-infra.labels" -}}
helm.sh/chart: {{ include "vuhive-cloud-infra.name" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: vuhive-cloud
{{- end }}

{{/*
OpenAPI Viewer fullname
*/}}
{{- define "vuhive-cloud-infra.openapiViewer.fullname" -}}
{{- printf "%s-openapi-viewer" (include "vuhive-cloud-infra.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
OpenAPI Viewer labels
*/}}
{{- define "vuhive-cloud-infra.openapiViewer.labels" -}}
{{ include "vuhive-cloud-infra.labels" . }}
app.kubernetes.io/name: {{ include "vuhive-cloud-infra.name" . }}-openapi-viewer
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: openapi-viewer
{{- end }}

{{/*
OpenAPI Viewer selector labels
*/}}
{{- define "vuhive-cloud-infra.openapiViewer.selectorLabels" -}}
app.kubernetes.io/name: {{ include "vuhive-cloud-infra.name" . }}-openapi-viewer
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Keycloak fullname
*/}}
{{- define "vuhive-cloud-infra.keycloak.fullname" -}}
{{- printf "%s-keycloak" (include "vuhive-cloud-infra.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Keycloak labels
*/}}
{{- define "vuhive-cloud-infra.keycloak.labels" -}}
{{ include "vuhive-cloud-infra.labels" . }}
app.kubernetes.io/name: {{ include "vuhive-cloud-infra.name" . }}-keycloak
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: keycloak
{{- end }}

{{/*
Keycloak selector labels
*/}}
{{- define "vuhive-cloud-infra.keycloak.selectorLabels" -}}
app.kubernetes.io/name: {{ include "vuhive-cloud-infra.name" . }}-keycloak
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
