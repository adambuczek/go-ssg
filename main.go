package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"gopkg.in/yaml.v3"
)

var md = goldmark.New(
	goldmark.WithExtensions(
		extension.Footnote,
		extension.DefinitionList,
	),
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
		parser.WithAttribute(),
	),
)

type Config struct {
	Src            string
	Dist           string
	Layouts        string
	Assets         string
	PollingTimeout time.Duration
}

var (
	config Config
	dev    bool
)

const (
	defaultPollingTimeout = 1 * time.Second
)

func loadConfig() Config {
	path := "config.yaml"
	file, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("error reading config file: %v", err)
	}
	var output Config
	err = yaml.Unmarshal(file, &output)
	if err != nil {
		log.Fatalf("error parsing config file: %v", err)
	}
	if output.Src == "" {
		log.Fatal("config: src is required")
	}
	if output.Dist == "" {
		log.Fatal("config: dist is required")
	}
	if output.Layouts == "" {
		log.Fatal("config: layouts is required")
	}
	if output.Assets == "" {
		log.Fatal("config: assets is required")
	}
	if output.PollingTimeout == 0 {
		output.PollingTimeout = defaultPollingTimeout
	}
	return output
}

type Meta struct {
	Layout      string
	Title       string
	Description string
	Date        time.Time
	Tags        []string
	Published   bool
	Excerpt     string
	Image       string
}

type Collections map[string][]Page

type Page struct {
	Meta        Meta
	Body        template.HTML
	OriginPath  string
	Collections Collections
	URL         string
}

func main() {
	flag.BoolVar(
		&dev,
		"dev",
		false,
		"enable to start a server with hot reload that watches source directory recursively",
	)
	flag.Parse()
	config = loadConfig()
	if err := build(); err != nil {
		log.Fatal(err)
	}
	if dev {
		go watch()

		// dev server
		if err := serve("8080"); err != nil {
			log.Fatal(err)
		}
	}
}

// build utils
func build() error {
	err := os.RemoveAll(config.Dist)
	if err != nil {
		return err
	}

	markdownFiles, err := findMarkdownFiles(config.Src)
	if err != nil {
		return err
	}

	var pages []Page

	for _, path := range markdownFiles {
		file, err := os.ReadFile(path)
		if err != nil {
			log.Printf("skipping %s: %v", path, err)
			continue
		}

		rawMeta, rawBody, err := splitMeta(file)
		if err != nil {
			log.Printf("skipping %s: %v", path, err)
			continue
		}

		meta, err := parseMeta(rawMeta)
		if err != nil {
			log.Printf("skipping %s: %v", path, err)
			continue
		}

		body, err := renderMarkdown(rawBody)
		if err != nil {
			log.Printf("skipping %s: %v", path, err)
			continue
		}

		page := Page{
			Meta:       meta,
			Body:       template.HTML(body),
			OriginPath: path,
		}

		page.URL = "/" + strings.TrimPrefix(
			strings.TrimSuffix(
				page.OriginPath,
				".md",
			),
			config.Src+"/") + "/"

		pages = append(pages, page)
	}

	collections := buildCollections(pages)

	for _, page := range pages {
		page.Collections = collections

		layout := filepath.Join(config.Layouts, page.Meta.Layout)

		renderedPage, err := renderPage(page, layout)
		if err != nil {
			log.Printf("skipping %s: %v", page.OriginPath, err)
			continue
		}

		if dev {
			renderedPage = injectReloadScript(renderedPage)
		}

		output := outputPath(page.OriginPath)

		err = writeFile(output, renderedPage)
		if err != nil {
			return err
		}
	}

	err = copyAssets()
	if err != nil {
		return err
	}

	return nil
}

func splitMeta(file []byte) ([]byte, []byte, error) {
	const frontMatterSplitter = "---\n"
	prefix := []byte(frontMatterSplitter)
	if !bytes.HasPrefix(file, prefix) {
		return nil, file, fmt.Errorf("file does not start with front matter")
	}
	file = bytes.TrimPrefix(file, prefix)
	meta, content, found := bytes.Cut(file, prefix)
	if !found {
		return nil, file, fmt.Errorf("front matter is not closed")
	}
	return meta, content, nil
}

func parseMeta(yamlData []byte) (Meta, error) {
	var meta Meta
	err := yaml.Unmarshal(yamlData, &meta)
	return meta, err
}

func renderMarkdown(content []byte) ([]byte, error) {
	var output bytes.Buffer
	err := md.Convert(content, &output)
	return output.Bytes(), err
}

func renderPage(page Page, templatePath string) ([]byte, error) {
	var output bytes.Buffer
	layout, err := template.New(
		filepath.Base(templatePath),
	).Funcs(
		template.FuncMap{
			"formatDate":            formatDate,
			"sortByDateDescInPlace": sortByDateDescInPlace,
			"renderMarkdown":        renderInlineMarkdown,
		}).ParseGlob(filepath.Join(config.Layouts, "*.html"))
	if err != nil {
		return nil, fmt.Errorf("error parsing layout file: %v", err)
	}
	err = layout.Execute(&output, page)
	if err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func findMarkdownFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".md") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func outputPath(path string) string {
	sourceTrimmed := strings.TrimPrefix(path, config.Src)
	extTrimmed := strings.TrimSuffix(sourceTrimmed, ".md")
	if filepath.Base(extTrimmed) == "index" {
		return filepath.Join(config.Dist, extTrimmed+".html")
	}
	return filepath.Join(config.Dist, extTrimmed, "index.html")
}

func writeFile(path string, data []byte) error {
	dirname := filepath.Dir(path)
	err := os.MkdirAll(dirname, fs.FileMode(0o755))
	if err != nil {
		return err
	}
	err = os.WriteFile(path, data, fs.FileMode(0o644))
	if err != nil {
		return err
	}
	return nil
}

func buildCollections(pages []Page) Collections {
	collections := Collections{}
	for _, page := range pages {
		for _, tag := range page.Meta.Tags {
			collections[tag] = append(collections[tag], page)
		}
	}
	return collections
}

func copyAssets() error {
	target := filepath.Join(config.Dist, filepath.Base(config.Assets))
	err := os.MkdirAll(target, fs.FileMode(0o755))
	if err != nil {
		return err
	}
	err = os.CopyFS(target, os.DirFS(config.Assets))
	if err != nil {
		return err
	}
	return nil
}

// template functions
func formatDate(t time.Time) string {
	return t.Format("January 2, 2006")
}

func sortByDateDescInPlace(pages []Page) []Page {
	sort.Slice(pages, func(i, j int) bool {
		return pages[i].Meta.Date.After(pages[j].Meta.Date)
	})
	return pages
}

func renderInlineMarkdown(s string) template.HTML {
	rendered, err := renderMarkdown([]byte(s))
	if err != nil {
		log.Printf("problem rendering inline markdown: %s", err)
	}
	return template.HTML(string(rendered))
}

// dev server
var reloadCh = make(chan struct{}, 1)

func watch() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	filepath.WalkDir(config.Src, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			watcher.Add(path)
		}
		return nil
	})

	timer := time.NewTimer(0)
	<-timer.C

	for {
		select {
		case _, ok := <-watcher.Events:
			if !ok {
				return
			}
			timer.Reset(config.PollingTimeout)

		case <-timer.C:
			log.Println("rebuilding...")
			if err := build(); err != nil {
				log.Println("build error:", err)
			}
			select {
			case reloadCh <- struct{}{}:
			default:
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println(err)
		}
	}
}

func sseHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	select {
	case <-reloadCh:
		fmt.Fprintf(w, "data: reload\n\n")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	case <-r.Context().Done():
	}
}

func serve(port string) error {
	addr := ":" + port
	http.Handle("/", http.FileServer(http.Dir(config.Dist)))
	http.HandleFunc("/_reload", sseHandler)
	log.Printf("serving at http://localhost%s", addr)
	return http.ListenAndServe(addr, nil)
}

func injectReloadScript(html []byte) []byte {
	anchorPoint := []byte("</body>")
	script := []byte(`
		<script> 
			new EventSource('/_reload').onmessage=()=>location.reload();
		</script> 
	`)
	return bytes.Replace(html, anchorPoint, append(script, anchorPoint...), 1)
}
