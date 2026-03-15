package main

import (
	"log"

	"camopanel/server/internal/app"
	"camopanel/server/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	instance, err := app.New(cfg)
	if err != nil {
		log.Fatalf("init app: %v", err)
	}

	if err := instance.Run(); err != nil {
		log.Fatalf("run server: %v", err)
	}
}
