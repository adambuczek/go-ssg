# go-ssg

A minimal static site generator written in Go. Converts Markdown files with YAML front matter into HTML using Go templates.

## Usage

```
go run . [-dev]
```

`-dev` starts a local server with file watching and hot reload.

Expects a `config.yaml` in the working directory.

## Config

```yaml
src: src
dist: dist
layouts: src/_layouts
assets: src/assets
polling_timeout: 1s   # optional, default 1s
port: "8080"          # optional, default 8080
```

| Field             | Required | Description                                      |
|-------------------|----------|--------------------------------------------------|
| `src`             | yes      | Source directory containing Markdown files       |
| `dist`            | yes      | Output directory (cleared on each build)         |
| `layouts`         | yes      | Directory containing HTML layout templates       |
| `assets`          | yes      | Directory copied verbatim into dist              |
| `polling_timeout` | no       | Debounce delay between file change and rebuild   |
| `port`            | no       | Port for the dev server                          |

## Directory structure

```
.
├── config.yaml
└── src/
    ├── index.md
    ├── notes.md
    ├── notes/
    │   └── my-post.md
    ├── _layouts/
    │   ├── index.html
    │   ├── notes.html
    │   └── note.html
    └── assets/
        └── style.css
```

Output mirrors the source structure under `dist/`. Each Markdown file becomes `index.html` inside a directory matching its name, except files already named `index.md`.

## Front matter

Every Markdown file must start with a YAML front matter block:

```yaml
---
layout: note
title: My Post
date: 2024-01-15
tags:
  - notes
---
```

| Field         | Description                                      |
|---------------|--------------------------------------------------|
| `layout`      | Template filename (with extension) to use       |
| `title`       | Page title                                       |
| `description` | Short description                                |
| `date`        | Publication date                                 |
| `tags`        | List of tags — used to build collections         |
| `published`   | Boolean — available in templates                 |
| `excerpt`     | Short excerpt — available in templates           |
| `image`       | Image path — available in templates              |

## Templates

Layouts are Go `html/template` files. Each file's name is the template name referenced by the `layout` front matter field.

Available template functions:

| Function          | Signature                          | Description                        |
|-------------------|------------------------------------|------------------------------------|
| `formatDate`      | `time.Time → string`               | Formats as `January 2, 2006`       |
| `sortByDateDesc`  | `[]*Page → []*Page`                | Returns pages sorted newest first  |
| `renderMarkdown`  | `string → template.HTML`           | Renders inline Markdown            |

Data available in every template:

```
.Meta.Title
.Meta.Date
.Meta.Tags
.Meta.Description
.Meta.Published
.Meta.Excerpt
.Meta.Image
.Body           — rendered HTML of the page body
.URL            — absolute URL path of the page
.Collections    — map of tag name → []*Page
```
