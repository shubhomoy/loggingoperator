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

{{- range .Inputs}}
[FILTER]
	Name       parser
	Match      {{ .Tag }}.*
	Key_Name   log
	Parser     {{ .Parser }}
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

[OUTPUT]
	Name            stdout
	Match           *
`
