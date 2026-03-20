{{/* Backend fullname */}}
{{- define "incident-replay.backend.fullname" -}}
{{- printf "%s-backend" .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/* Frontend fullname */}}
{{- define "incident-replay.frontend.fullname" -}}
{{- printf "%s-frontend" .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/* Common labels */}}
{{- define "incident-replay.labels" -}}
app.kubernetes.io/name: {{ include "incident-replay.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "incident-replay.name" -}}
incident-replay
{{- end -}}
