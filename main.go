package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"

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
	file, err := readFile("test.md")
	if err != nil {
		log.Fatal(err)
	}

	rawMeta, rawBody, err := splitMeta(file)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(string(rawBody))

	meta, err := parseMeta(rawMeta)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%+v", meta)

	body, err := renderBody(rawBody)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%s", body)

	page := Page{
		Meta: meta,
		Body: template.HTML(body),
	}

	renderedPage, err := renderPage(page, "./template.html")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%s", renderedPage)
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
		return nil, err
	}
	layout.Execute(&output, page)
	return output.Bytes(), nil
}
