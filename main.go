package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"

	H "github.com/autovia/tri/handlers"
	S "github.com/autovia/tri/structs"
)

func main() {
	app := &S.App{}
	app.Addr = flag.String("addr", ":3000", "TCP address for the server to listen on, in the form host:port")
	app.AccessKey = flag.String("access-key", "user", "aws_access_key_id")
	app.SecretKey = flag.String("secret-key", "password", "aws_secret_access_key")
	app.Mount = flag.String("mount", "./mount", "root directory containing the buckets and files")
	flag.Parse()

	// Router
	app.Router = http.NewServeMux()
	app.Router.Handle("/", S.Auth{App: app, R: map[string]any{
		"GET":    H.Get,
		"PUT":    H.Put,
		"POST":   H.Post,
		"DELETE": H.Delete,
		"HEAD":   H.Head,
	}})

	// Check fs folders
	if _, err := os.Stat(*app.Mount); os.IsNotExist(err) {
		if err := os.Mkdir(*app.Mount, os.ModePerm); err != nil {
			log.Fatalf("Can not create storage directoy at %s", *app.Mount)
		}
		log.Printf("Storage directory created at %s", *app.Mount)
	}

	metadata := filepath.Join(*app.Mount, H.Metadata)
	if _, err := os.Stat(metadata); os.IsNotExist(err) {
		if err := os.Mkdir(metadata, os.ModePerm); err != nil {
			log.Fatalf("Can not create metadata directoy at %s", *app.Mount)
		}
		log.Printf("Metadata directory created at %s", metadata)
	}

	// Server
	srv := &http.Server{
		Addr:    *app.Addr,
		Handler: app.Router,
		//TLSConfig:    cfg,
		//TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}
	log.Printf("Listen on %s", *app.Addr)
	log.Fatal(srv.ListenAndServe())
}
