package main

type entryFormatter interface {
	entryTmpl() string
	listTmpl() string
}

type layoutFormatter interface {
	layoutTmpl() string
}

type textFormatter struct{}

func (f *textFormatter) entryTmpl() string {
	return `
{{- define "entry" }}
{{ .Request }}
**********************************************************
{{- range .Responses }}
{{ .Lang }}:
{{ range $idx, $tr := .Translations -}}
{{ inc $idx }}. {{ $tr }}
{{ end -}}
----------------------------------------------------------
{{- end }}
{{- end }}`
}

func (f *textFormatter) listTmpl() string {
	return `
{{- range .Entries -}}
{{.Request}}
{{ end }}
**********************************************************
{{ range .Entries -}}
{{ template "entry" . }}
{{ end }}
`
}

type htmlFormatter struct{}

func (f *htmlFormatter) entryTmpl() string {
	return `{{ define "entry" -}}
	<dt id={{ inc .idx }}>{{ .entry.Request }}</dt>
	{{ range .entry.Responses }}
	<dd>
		<header>{{ .Lang }}</header>
		<ol>
			{{ range .Translations -}}
			<li>{{ . }}</li>
			{{ end }}
		</ol>
	</dd>
	{{- end }}
	{{- end }}`
}

func (f *htmlFormatter) listTmpl() string {
	return `{{ define "list" }}
<ul>
	{{- range $idx, $entry := .Entries }}
	<li><a href="#{{ inc $idx }}">{{ $entry.Request }}</a></li>
	{{- end }}
</ul>

<dl>
	{{ range $idx, $entry := .Entries }}
	{{ template "entry" dict "idx" $idx "entry" $entry }}
	{{ end }}
</dl>
{{ end }}`
}

func (f *htmlFormatter) layoutTmpl() string {
	return `
<html>
<head>
<meta charset="utf-8">
</head>
<body>
	{{ template "list" . }}
</body>
</html>`
}

func inc(i int) int {
	return i + 1
}

func dict(values ...interface{}) map[string]interface{} {
	d := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		d[values[i].(string)] = values[i+1]
	}
	return d
}
