package main

import (
	"log"
	"net/http"
	"os"

	"tuxpa.in/a/pprofweb/pprofweb"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/spf13/afero"
)

func main() {

	filepath := os.Getenv("STORAGE_PATH")
	if filepath == "" {
		filepath = "./tmp"
	}

	s := pprofweb.NewServer(
		afero.NewBasePathFs(afero.NewOsFs(), filepath),
		pprofweb.Config{},
	)

	port := os.Getenv("PORT")
	if port == "" {
		port = "7443"
		log.Printf("warning: %s not specified; using default %s", "PORT", port)
	}

	addr := ":" + port
	log.Printf("listen addr %s (http://localhost:%s/)", addr, port)
	r := chi.NewRouter()
	r.Use(middleware.Recoverer, middleware.Logger)
	r.Use(middleware.NewCompressor(6).Handler)
	r.Route("/", s.HandleHTTP())
	if err := http.ListenAndServe(addr, r); err != nil {
		panic(err)
	}
}
