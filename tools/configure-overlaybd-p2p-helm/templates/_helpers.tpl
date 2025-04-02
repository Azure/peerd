{{- define "overlaybd.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "overlaybd.namespace" -}}
{{- if .Values.overlaybd.namespace.k8s }}
{{- .Values.overlaybd.namespace.k8s }}
{{- else }}
{{ include "overlaybd.name" . }}-ns
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "overlaybd.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "overlaybd.labels" -}}
helm.sh/chart: {{ include "overlaybd.chart" . }}
{{ include "overlaybd.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Common selector labels
*/}}
{{- define "overlaybd.selectorLabels" -}}
app: {{ include "overlaybd.name" . }}
app.kubernetes.io/name: {{ include "overlaybd.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}