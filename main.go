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

	"github.com/adambuczek/go-ssg/config"
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

type Collections map[string][]*Page

type Page struct {
	Meta        Meta
	Body        template.HTML
	OriginPath  string
	Collections Collections
	URL         string
}

func main() {
	var dev bool
	flag.BoolVar(
		&dev,
		"dev",
		false,
		"enable to start a server with hot reload that watches source directory recursively",
	)
	flag.Parse()

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	if err := build(cfg, dev); err != nil {
		log.Fatal(err)
	}
	if dev {
		go watch(cfg)

		// dev server
		if err := serve(cfg); err != nil {
			log.Fatal(err)
		}
	}
}

// build utils
func build(cfg config.Config, dev bool) error {
	err := os.RemoveAll(cfg.Dist)
	if err != nil {
		return err
	}

	markdownFiles, err := findMarkdownFiles(cfg.Src)
	if err != nil {
		return err
	}

	var pages []*Page

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
			cfg.Src+"/") + "/"

		pages = append(pages, &page)
	}

	collections := buildCollections(pages)

	layouts, err := template.New("").Funcs(
		template.FuncMap{
			"formatDate":     formatDate,
			"sortByDateDesc": sortByDateDesc,
			"renderMarkdown": renderInlineMarkdown,
		}).ParseGlob(filepath.Join(cfg.Layouts, "*.html"))
	if err != nil {
		return fmt.Errorf("error when parsing layout tempaltes: %v", err)
	}

	for _, page := range pages {
		page.Collections = collections

		renderedPage, err := renderPage(*page, layouts)
		if err != nil {
			log.Printf("skipping %s: %v", page.OriginPath, err)
			continue
		}

		if dev {
			renderedPage = injectReloadScript(renderedPage)
		}

		output := outputPath(page.OriginPath, cfg)

		err = writeFile(output, renderedPage)
		if err != nil {
			return err
		}
	}

	err = copyAssets(cfg)
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

func renderPage(page Page, layouts *template.Template) ([]byte, error) {
	var output bytes.Buffer
	err := layouts.ExecuteTemplate(&output, page.Meta.Layout, page)
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

func outputPath(path string, cfg config.Config) string {
	sourceTrimmed := strings.TrimPrefix(path, cfg.Src)
	extTrimmed := strings.TrimSuffix(sourceTrimmed, ".md")
	if filepath.Base(extTrimmed) == "index" {
		return filepath.Join(cfg.Dist, extTrimmed+".html")
	}
	return filepath.Join(cfg.Dist, extTrimmed, "index.html")
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

func buildCollections(pages []*Page) Collections {
	collections := Collections{}
	for _, page := range pages {
		for _, tag := range page.Meta.Tags {
			collections[tag] = append(collections[tag], page)
		}
	}
	return collections
}

func copyAssets(cfg config.Config) error {
	target := filepath.Join(cfg.Dist, filepath.Base(cfg.Assets))
	err := os.MkdirAll(target, fs.FileMode(0o755))
	if err != nil {
		return err
	}
	err = os.CopyFS(target, os.DirFS(cfg.Assets))
	if err != nil {
		return err
	}
	return nil
}

// template functions
func formatDate(t time.Time) string {
	return t.Format("January 2, 2006")
}

func sortByDateDesc(pages []*Page) []*Page {
	sorted := append([]*Page{}, pages...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Meta.Date.After(sorted[j].Meta.Date)
	})
	return sorted
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

func watch(cfg config.Config) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	filepath.WalkDir(cfg.Src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("problem adding %s to watched directories: %s", path, err)
			return filepath.SkipDir
		}
		if d.IsDir() {
			err := watcher.Add(path)
			if err != nil {
				log.Printf("problem adding %s to watched directories: %s", path, err)
			}
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
			timer.Reset(cfg.PollingTimeout)

		case <-timer.C:
			log.Println("rebuilding...")
			if err := build(cfg, true); err != nil {
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

func serve(cfg config.Config) error {
	addr := ":" + cfg.Port
	http.Handle("/", http.FileServer(http.Dir(cfg.Dist)))
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
