package main

import (
	"github.com/gomarkdown/markdown"
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
  <link rel="stylesheet" href="/css/main.css" >
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
		log.Fatal(err)
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

	contents, err := ioutil.ReadFile(path)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if isPage {
		body := string(markdown.ToHTML(contents, nil, nil))
		title := getPageTitle(path)

		a.sendPage(w, &PageTemplateBindings{Body: body, Title: title})
	} else {
		w.Write(contents)
	}
}

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("usage: %s <site root>", os.Args[0])
	}

	a := App{}

	t, err := template.New("page").Parse(pageTemplate)
	if err != nil {
		log.Fatal(err)
	}

	a.root = os.Args[1]
	a.t = t
	http.HandleFunc("/", a.serve)

	log.Fatal(http.ListenAndServe(":8000", nil))
}
