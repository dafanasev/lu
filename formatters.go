package main

type entryFormatter interface {
	entryTmpl() string
	listTmpl() string
}

type textFormatter struct{}

func (f *textFormatter) entryTmpl() string {
	return `
{{- define "entry" }}
{{.Request}}
**********************************************************
{{- range .Responses }}
{{.Lang}}:
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
	return `{{- define "entry" }}
	<dt>{{.Request}}</dt>
	{{ range .Responses -}}
	<dd>
		<header>{{.Lang}}</header>
		<ul>
			{{ range $idx, $tr := .Translations -}}
			<li>{{ inc $idx }}. {{ $tr }}</li>
			{{ end }}
		</ul>
	</dd>
	{{ end -}}
	{{ end }}`
}

func (f *htmlFormatter) listTmpl() string {
	return `<ul>{{range .Entries}}
	<li>{{.Request}}</li>{{end}}
</ul>

<dl>
	{{- range .Entries }}
	{{ template "entry" . }}
	{{- end }}
</dl>
`
}

func inc(i int) int {
	return i + 1
}
