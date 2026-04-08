package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	"gopkg.in/yaml.v3"
)

type Meta struct {
	Layout    string
	Title     string
	Date      string
	Tags      []string
	Published bool
	Excerpt   string
	Image     string
}

type Page struct {
	Meta Meta
	Body template.HTML
}

func main() {
	markdownFiles, err := findMarkdownFiles("src/")
	if err != nil {
		log.Fatal(err)
	}

	for _, path := range markdownFiles {
		file, err := readFile(path)
		if err != nil {
			log.Fatal(err)
		}

		rawMeta, rawBody, err := splitMeta(file)
		if err != nil {
			log.Fatalf("error when processing %s: %v", path, err)
		}

		meta, err := parseMeta(rawMeta)
		if err != nil {
			log.Fatal(err)
		}

		body, err := renderBody(rawBody)
		if err != nil {
			log.Fatal(err)
		}

		page := Page{
			Meta: meta,
			Body: template.HTML(body),
		}

		layout := layoutPath(page.Meta.Layout)

		renderedPage, err := renderPage(page, layout)
		if err != nil {
			log.Fatalf("error when processing %s: %v", path, err)
		}

		output := outputPath(path)

		err = writeFile(output, renderedPage)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func readFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	return data, err
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

func renderBody(content []byte) ([]byte, error) {
	var output bytes.Buffer
	md := goldmark.New()
	err := md.Convert(content, &output)
	return output.Bytes(), err
}

func renderPage(page Page, templatePath string) ([]byte, error) {
	var output bytes.Buffer
	layout, err := template.ParseFiles(templatePath)
	if err != nil {
		return nil, fmt.Errorf("error parsing laout file: %v", err)
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
	sourceTrimmed := strings.TrimPrefix(path, "src/")
	extTrimmed := strings.TrimSuffix(sourceTrimmed, ".md")
	return filepath.Join("dist", extTrimmed, "index.html")
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

func layoutPath(layout string) string {
	layoutsRoot := "./src/layouts"
	return filepath.Join(layoutsRoot, layout)
}
