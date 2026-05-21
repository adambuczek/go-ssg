// Package server creates a dev server with watcher
package server

import (
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/adambuczek/go-ssg/config"
	"github.com/fsnotify/fsnotify"
)

type Server struct {
	reloadCh chan struct{}
}

func New() *Server {
	return &Server{reloadCh: make(chan struct{}, 1)}
}

func (s *Server) Watch(src string, timeout time.Duration, build func() error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close() //nolint:errcheck

	err = filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
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
	if err != nil {
		log.Printf("problem walking directories: %s", err)
	}

	// drain inital zero duration tick to prevent immidiate trigger
	timer := time.NewTimer(0)
	<-timer.C

	log.Printf("polling file changes every %s", timeout)

	for {
		select {
		case _, ok := <-watcher.Events:
			if !ok {
				return
			}
			timer.Reset(timeout)

		case <-timer.C:
			log.Println("rebuilding...")
			if err := build(); err != nil {
				log.Println("build error:", err)
			}
			select {
			case s.reloadCh <- struct{}{}:
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

func (s *Server) sseHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	select {
	case <-s.reloadCh:
		if _, err := fmt.Fprintf(w, "data: reload\n\n"); err != nil {
			return
		}
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	case <-r.Context().Done():
	}
}

func (s *Server) Serve(portNumber int, dist string) error {
	port := strconv.Itoa(portNumber)
	_, err := net.LookupPort("tcp", port)
	if err != nil {
		return err
	}
	addr := ":" + port
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(dist)))
	mux.HandleFunc(config.ReloadEndpoint, s.sseHandler)
	log.Printf("serving at http://localhost%s", addr)
	return http.ListenAndServe(addr, mux)
}
