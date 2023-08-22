package template

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"strings"
)

type Option = func(*templates)

type Template interface {
	Render(w io.Writer, name string, data any) error
}

func FS(fs fs.FS) Option {
	return func(t *templates) {
		t.assets = fs
		t.funcMap = make(template.FuncMap)
	}
}

func FuncMap(funcs template.FuncMap) Option {
	return func(t *templates) {
		for name, fn := range funcs {
			t.funcMap[name] = fn
		}
	}
}

func Func(name string, fn any) Option {
	return func(t *templates) {
		t.funcMap[name] = fn
	}
}

func New(options ...Option) Template {
	t := &templates{}

	for _, fn := range options {
		fn(t)
	}

	if t.assets == nil {
		panic(`nil fs.FS`)
	}

	t.template = template.Must(
		load(t.assets, `page`, t.funcMap, template.Must(
			load(t.assets, `component`, t.funcMap, template.Must(
				load(t.assets, `layout`, t.funcMap, nil),
			)),
		)))

	return t
}

type templates struct {
	assets   fs.FS
	funcMap  template.FuncMap
	template *template.Template
}

func (t templates) Render(w io.Writer, name string, data any) error {
	template := t.template.Lookup(name)
	if template == nil {
		return fmt.Errorf(`template %q not found`, name)
	}

	if strings.HasPrefix(name, `component/`) {
		return template.Execute(w, data)
	}

	return template.ExecuteTemplate(w, `base`, data)
}

func load(assets fs.FS, base string, funcMap template.FuncMap, components *template.Template) (*template.Template, error) {
	root := template.New(base)

	if components != nil {
		root = template.Must(components.Clone()).New(base)
	}

	err := fs.WalkDir(assets, base, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		data, err := fs.ReadFile(assets, path)
		if err != nil {
			return err
		}

		name := templateName(base, path)
		_, err = root.New(name).Funcs(funcMap).Parse(string(data))

		return err
	})

	if err != nil {
		return nil, err
	}

	return root, nil
}

func templateName(base, path string) string {
	if strings.Contains(path, `.`) {
		return strings.Split(path, `.`)[0]
	}

	return path
}
