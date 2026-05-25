{{- define "todo-app.name" -}}
{{- .Chart.Name }}
{{- end }}

{{- define "todo-app.fullname" -}}
{{- printf "%s" .Release.Name }}
{{- end }}

{{- define "todo-app.labels" -}}
app.kubernetes.io/name: {{ include "todo-app.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}

{{- define "todo-app.selectorLabels" -}}
app.kubernetes.io/name: {{ include "todo-app.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Defines the DB host: if subchart is enabled — subchart service name, otherwise — db.host from values
*/}}
{{- define "todo-app.dbHost" -}}
{{- if .Values.postgresql.enabled -}}
{{- printf "%s-postgresql" .Release.Name -}}
{{- else -}}
{{- .Values.db.host -}}
{{- end -}}
{{- end }}
