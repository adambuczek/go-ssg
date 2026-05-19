// Package builder builds the source md files into html and moves assets
package builder

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/adambuczek/go-ssg/config"
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

func Build(cfg config.Config, dev bool) error {
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

	layouts, err := template.New("_root").Funcs(
		template.FuncMap{
			"formatDate":     formatDate,
			"sortByDateDesc": sortByDateDesc,
			"renderMarkdown": renderInlineMarkdown,
		}).ParseGlob(filepath.Join(cfg.Layouts, "*.html"))
	if err != nil {
		return fmt.Errorf("error when parsing layout templates: %v", err)
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
	sorted := slices.Clone(pages)
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

func injectReloadScript(html []byte) []byte {
	anchorPoint := []byte("</body>")
	script := fmt.Appendf([]byte{}, `
		<script> 
			new EventSource('%s').onmessage=()=>location.reload();
		</script> 
	`, config.ReloadEndpoint)
	return bytes.Replace(html, anchorPoint, append(script, anchorPoint...), 1)
}
