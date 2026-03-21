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
