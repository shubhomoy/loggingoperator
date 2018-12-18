package fluentbit

var configmapTemplate = `
[SERVICE]
	Flush         1
	Log_Level     debug
	Log_File      {{ .FluentBitLogFile }}
	Daemon        off
	HTTP_Server   On
	HTTP_Listen   0.0.0.0
	HTTP_Port     2020
	Parsers_File  parsers.conf


{{- range .Inputs}}
[INPUT]
	Name              tail
	Tag               {{ .Tag }}.*
	Path              /var/log/containers/{{ .DeploymentName }}*
	Parser            json_parser
{{- end}}

{{- range $in := .Inputs}}
	{{- range $par := $in.Parsers}}

[FILTER]
	Name       parser
	Match      {{ $in.Tag }}.*
	Key_Name   log
	Parser     {{ $par.Name }}
	{{- end}}
{{- end}}


{{ if .K8sMetadata -}}
[FILTER]
	Name                kubernetes
	Match               **
	K8S-Logging.Parser  On
	Merge_Log           On
{{- end}}

[OUTPUT]
	Name            forward
	Match           *
	Host            ${FLUENTD_SERVICE_HOST}
	Port            ${FLUENTD_SERVICE_PORT}
`
