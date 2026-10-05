package main

import (
	"github.com/arfajhf/copytygo/config"
	"github.com/arfajhf/copytygo/core"
	"log"
	"time"
)

func main() {
	if err := config.LoadEnv("example/.env"); err != nil {
		log.Fatal(err)
	}
	app := core.New()
	app.Use(core.SecurityHeaders(), core.BodyLimit(2<<20), core.RateLimit(120, time.Minute))
	app.Get("/", func(ctx *core.Context) error {
		return ctx.JSON(core.Map{"framework": "CopyTyGo", "version": "1.0.0", "status": "running"})
	}).Name("home")
	api := app.Group("/api")
	api.Get("/health", func(ctx *core.Context) error { return ctx.JSON(core.Map{"ok": true}) }).Name("api.health")
	log.Fatal(app.Run())
}
