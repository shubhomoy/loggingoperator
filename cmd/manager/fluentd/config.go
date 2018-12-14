package fluentd

var configmapTemplate = `
<source>
	@type forward
	port 8777
	bind 0.0.0.0
</source>

<filter **>
	@type elasticsearch_timestamp_check
</filter>

{{- range .Inputs}}
<match {{ .Tag }}**>
	@type elasticsearch
	host "#{ENV['ES_HOST']}"
	port "#{ENV['ES_PORT']}"
	flush_interval 1s
	logstash_format true
	logstash_prefix {{ .Tag }}
</match>
{{- end}}
`
