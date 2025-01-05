package template

import (
	"bytes"
	"io/fs"
	"path/filepath"
	"strings"
	"testing/fstest"
	"text/template"
)

var funcMap = template.FuncMap{
	"escape": func(input string) string {
		return EscapeMarkdown(input)
	},
}

type TemplateRenderer struct {
	*template.Template
}

type Template interface {
	NewTemplate(templates fs.FS, name string, data any) (*string, TemplateRenderer, error)
}

// EscapeMarkdown escapes all Markdown special characters in a given string.
func EscapeMarkdown(text string) string {
	var escaped bytes.Buffer
	for _, char := range text {
		if strings.ContainsRune("\\`*_{}[]()#+-.!", char) {
			escaped.WriteRune('\\')
		}
		escaped.WriteRune(char)
	}
	return escaped.String()
}

func NewTemplate(templates fs.ReadFileFS, name string, data any) (*string, *template.Template, error) {
	return TemplateRenderer{}.NewTemplate(templates, name, data)
}

func New(tmpl string, name string, data any) (*string, *template.Template, error) {
	fs := &fstest.MapFS{
		name: &fstest.MapFile{Data: []byte(tmpl)},
	}
	return TemplateRenderer{}.NewTemplate(fs, name, data)
}

func (TemplateRenderer) NewTemplate(templates fs.ReadFileFS, name string, data any) (*string, *template.Template, error) {
	b, err := templates.ReadFile(name)

	if err != nil {
		return nil, nil, err
	}

	tmpl := template.New(filepath.Base(name))

	tmpl.Funcs(funcMap)

	tmpl, err = tmpl.Parse(string(b))
	if err != nil {
		return nil, nil, err
	}

	buf := new(bytes.Buffer)

	err = tmpl.Execute(buf, data)
	if err != nil {
		return nil, nil, err
	}
	str := buf.String()
	return &str, tmpl, nil
}
