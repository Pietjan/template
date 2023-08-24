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

	t.template = template.Must(load(t.assets, list(t.assets, `component`), t.funcMap,
		template.Must(load(t.assets, list(t.assets, `layout`), t.funcMap, nil))),
	)

	for _, page := range list(t.assets, `page`) {
		t.page = append(t.page, template.Must(load(t.assets, []string{page}, t.funcMap, t.template)))
	}

	return t
}

type templates struct {
	assets   fs.FS
	funcMap  template.FuncMap
	template *template.Template
	page     []*template.Template
}

func (t templates) Render(w io.Writer, name string, data any) error {
	var template *template.Template

	if strings.HasPrefix(name, `page/`) {
		for _, page := range t.page {
			template = page.Lookup(name)
			if template != nil {
				break
			}
		}
	} else {
		template = t.template.Lookup(name)
	}

	if template == nil {
		return fmt.Errorf(`template %q not found`, name)
	}

	if strings.HasPrefix(name, `component/`) || strings.HasPrefix(name, `layout/`) {
		return template.Execute(w, data)
	}

	return template.ExecuteTemplate(w, `base`, data)
}

func list(assets fs.FS, base string) (paths []string) {
	_ = fs.WalkDir(assets, base, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		paths = append(paths, path)

		return nil
	})

	return
}

func load(assets fs.FS, paths []string, funcMap template.FuncMap, components *template.Template) (*template.Template, error) {
	root := template.New(``)

	if components != nil {
		root = template.Must(components.Clone()).New(``)
	}

	for _, path := range paths {
		data, err := fs.ReadFile(assets, path)
		if err != nil {
			return nil, err
		}

		name := templateName(path)
		_, err = root.New(name).Funcs(funcMap).Parse(string(data))
	}

	return root, nil
}

func templateName(path string) string {
	if strings.Contains(path, `.`) {
		return strings.Split(path, `.`)[0]
	}

	return path
}
