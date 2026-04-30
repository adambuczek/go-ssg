package main

import (
	"flag"
	"log"

	"github.com/adambuczek/go-ssg/builder"
	"github.com/adambuczek/go-ssg/config"
	"github.com/adambuczek/go-ssg/server"
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
		go server.Watch(cfg.Src, cfg.PollingTimeout, func() error {
			return builder.Build(cfg, true)
		})

		// dev server
		if err := server.Serve(cfg.Port, cfg.Dist); err != nil {
			log.Fatal(err)
		}
	}
}
