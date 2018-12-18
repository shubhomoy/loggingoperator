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


{{- range $in := .Inputs}}
<match {{ $in.Tag }}**>
	@type copy
	{{- range $out := $in.Outputs}}
	{{ if eq $out.Type "elasticsearch" -}}
	<store>
		@type elasticsearch
		host "#{ENV['ES_HOST']}"
		port "#{ENV['ES_PORT']}"
		flush_interval 1s
		logstash_format true
		logstash_prefix {{ $out.IndexPattern }}
	</store>
	{{- end}}
	{{if ne $out.Type "elasticsearch" -}}
	<store>
		@type {{$out.Type}}
	</store>
{{- end}}
{{- end}}
</match>
{{- end}}
`
