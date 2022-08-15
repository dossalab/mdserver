package main

import (
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/parser"
	"github.com/snabb/sitemap"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

type App struct {
	t    *template.Template
	root string
	base string
	fs   http.Handler
}

type PageTemplateBindings struct {
	Body  string
	Title string
}

type SitemapEntry struct {
	Path    string
	LastMod time.Time
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

func findSitemapEntries(root string) []SitemapEntry {
	var entries []SitemapEntry

	err := filepath.WalkDir(root, func(syspath string, d os.DirEntry, err error) error {
		path, _ := filepath.Rel(root, syspath)
		if err != nil {
			return nil
		}

		ext := filepath.Ext(path)
		if ext == ".md" {
			fi, err := d.Info()
			if err != nil {
				return nil
			}

			entry := SitemapEntry{
				Path:    strings.TrimSuffix(path, ext),
				LastMod: fi.ModTime(),
			}

			entries = append(entries, entry)
		}

		return nil
	})

	if err != nil {
		log.Printf("unable to walk the root directory - %v", err)
	}

	return entries
}

func (a *App) generateSitemap(w http.ResponseWriter, r *http.Request) {
	sm := sitemap.New()
	entries := findSitemapEntries(a.root)

	for _, entry := range entries {
		sm.Add(&sitemap.URL{
			Loc:        path.Join(a.base, entry.Path),
			LastMod:    &entry.LastMod,
			ChangeFreq: sitemap.Weekly,
		})
	}

	sm.WriteTo(w)
}

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("usage: %s <name> <site root>", os.Args[0])
	}

	base := os.Args[1]
	root := os.Args[2]

	t, err := template.New("page").Parse(pageTemplate)
	if err != nil {
		log.Fatal(err)
	}

	a := App{
		root: root,
		base: base,
		t:    t,
		fs:   http.FileServer(http.Dir(root)),
	}

	http.HandleFunc("/", a.serve)
	http.HandleFunc("/sitemap.xml", a.generateSitemap)

	log.Fatal(http.ListenAndServe(":8000", nil))
}
