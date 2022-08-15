package main

import (
	"flag"
	"fmt"
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

type Settings struct {
	Root     string
	Port     uint
	Hostname string
	SiteName string
}

type App struct {
	t        *template.Template
	fs       http.Handler
	settings *Settings
}

type PageTemplateBindings struct {
	Body     string
	Title    string
	SiteName string
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
  <title>{{ .Title }} | {{ .SiteName }}</title>
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

	return path.Join(a.settings.Root, url), page
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

		a.sendPage(w, &PageTemplateBindings{
			Body: body,
			Title: title,
			SiteName: a.settings.SiteName,
		})
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
	entries := findSitemapEntries(a.settings.Root)

	for _, entry := range entries {
		sm.Add(&sitemap.URL{
			Loc:        path.Join(a.settings.Hostname, entry.Path),
			LastMod:    &entry.LastMod,
			ChangeFreq: sitemap.Weekly,
		})
	}

	sm.WriteTo(w)
}

func parseSettings() *Settings {
	port := flag.Uint("port", 8000, "port")
	hostname := flag.String("host", "http://example.com", "hostname (for sitemap.xml)")
	siteName := flag.String("sitename", "Example", "the name of the website (as shown in title)")
	root := flag.String("root", ".", "the base directory where the site is located")
	flag.Parse()

	return &Settings{
		Port:     *port,
		Root:     *root,
		Hostname: *hostname,
		SiteName: *siteName,
	}
}

func main() {
	settings := parseSettings()

	t, err := template.New("page").Parse(pageTemplate)
	if err != nil {
		log.Fatal(err)
	}

	a := App{
		t:        t,
		fs:       http.FileServer(http.Dir(settings.Root)),
		settings: settings,
	}

	http.HandleFunc("/", a.serve)
	http.HandleFunc("/sitemap.xml", a.generateSitemap)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", settings.Port), nil))
}
