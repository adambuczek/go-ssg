package main

import (
	"bytes"
	"fmt"
	"log"
	"os"

	"github.com/yuin/goldmark"
	"gopkg.in/yaml.v3"
)

type FrontMatter struct {
	Layout    string
	Title     string
	Date      string
	Tags      []string
	Published bool
	Excerpt   string
	Image     string
}

func main() {
	file, err := readFile("test.md")
	if err != nil {
		log.Fatal(err)
	}

	frontMatterRaw, content, err := splitFrontMatter(file)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(string(content))
	frontMatter, err := parseFrontMatter(frontMatterRaw)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%+v", frontMatter)

	html, err := renderMarkdown(content)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%s", html)
}

func readFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	return data, err
}

func splitFrontMatter(file []byte) ([]byte, []byte, error) {
	const frontMatterSplitter = "---\n"
	prefix := []byte(frontMatterSplitter)
	if !bytes.HasPrefix(file, prefix) {
		return nil, file, fmt.Errorf("file does not start with front matter")
	}
	file = bytes.TrimPrefix(file, prefix)
	matter, content, found := bytes.Cut(file, prefix)
	if !found {
		return nil, file, fmt.Errorf("front matter is not closed")
	}
	return matter, content, nil
}

func parseFrontMatter(data []byte) (FrontMatter, error) {
	var frontMatter FrontMatter
	err := yaml.Unmarshal(data, &frontMatter)
	return frontMatter, err
}

func renderMarkdown(content []byte) ([]byte, error) {
	md := goldmark.New()
	var output bytes.Buffer
	err := md.Convert(content, &output)
	return output.Bytes(), err
}
