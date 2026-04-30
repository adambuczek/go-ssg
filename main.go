package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/adambuczek/go-ssg/builder"
	"github.com/adambuczek/go-ssg/config"
	"github.com/fsnotify/fsnotify"
)

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

	if err := builder.Build(cfg, dev); err != nil {
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
			if err := builder.Build(cfg, true); err != nil {
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
