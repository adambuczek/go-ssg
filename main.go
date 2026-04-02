package main

import (
	"bytes"
	"fmt"
	"log"
	"os"

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

	matter, content, err := splitFrontMatter(file)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(string(content))
	parseFrontMatter(matter)
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

func parseFrontMatter(data []byte) {
	var frontMatter FrontMatter
	yaml.Unmarshal(data, &frontMatter)
	log.Printf("%+v", frontMatter)
}
