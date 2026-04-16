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
Определяет хост БД: если subchart включён — имя сервиса subchart, иначе — db.host из values
*/}}
{{- define "todo-app.dbHost" -}}
{{- if .Values.postgresql.enabled -}}
{{- printf "%s-postgresql" .Release.Name -}}
{{- else -}}
{{- .Values.db.host -}}
{{- end -}}
{{- end }}
