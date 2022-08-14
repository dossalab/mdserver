package main

import (
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/parser"
	"text/template"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"path"
	"strings"
	"io"
	"os"
)

type App struct {
	t *template.Template
	root string
	fs http.Handler
}

type PageTemplateBindings struct {
	Body  string
	Title string
}

const pageTemplate = `<!DOCTYPE HTML>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{ .Title }}</title>
  <link rel="stylesheet" type="text/css" href="/css/main.css" >
</head>

<body>
{{ .Body  }}
</body>

</html>`

func getPageTitle(url string) string {
	filename := strings.TrimSuffix(filepath.Base(url), filepath.Ext(url))
	return strings.Title(filename)
}

func (a *App) sendPage(w io.Writer, bindings *PageTemplateBindings) {
	err := a.t.Execute(w, bindings)
	if err != nil {
		log.Print(err)
	}
}

func (a *App) buildPath(url string) (string, bool) {
	page := false

	if url == "" {
		url = "index"
	}

	extension := filepath.Ext(url)
	if extension == "" {
		url += ".md"
		page = true
	}

	return path.Join(a.root, url), page
}

func (a *App) serve(w http.ResponseWriter, r *http.Request) {
	path, isPage := a.buildPath(r.URL.Path[1:])
	if isPage {
		contents, err := ioutil.ReadFile(path)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		extensions := parser.CommonExtensions | parser.Attributes
		parser := parser.NewWithExtensions(extensions)

		body := string(markdown.ToHTML(contents, parser, nil))
		title := getPageTitle(path)

		a.sendPage(w, &PageTemplateBindings{Body: body, Title: title})
	} else {
		a.fs.ServeHTTP(w, r)
	}
}

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("usage: %s <site root>", os.Args[0])
	}

	root := os.Args[1]

	t, err := template.New("page").Parse(pageTemplate)
	if err != nil {
		log.Fatal(err)
	}

	a := App{
		root: root,
		t: t,
		fs: http.FileServer(http.Dir(root)),
	}

	http.HandleFunc("/", a.serve)

	log.Fatal(http.ListenAndServe(":8000", nil))
}
