package fluentbit

var parsersTemplate = `
{{- range .Parsers}}
[PARSER]
	Name   {{ .Name }}
	Format regex
	Regex  {{ .Regex }}
{{- end}}

[PARSER]
	Name        json_parser
	Format      json
	Decode_Field_As    escaped     log
`
