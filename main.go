package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

const page = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Image Trust Demo</title>
  <style>
    :root { color-scheme: dark; font-family: system-ui, sans-serif; }
    body { min-height: 100vh; margin: 0; display: grid; place-items: center; background: #101828; }
    main { padding: 3rem 4rem; border: 1px solid #344054; border-radius: 1rem; background: #1d2939; box-shadow: 0 1rem 3rem #0005; }
    h1 { margin: 0; color: #7f56d9; font-size: clamp(2.5rem, 8vw, 5rem); }
  </style>
</head>
<body><main><h1>Hello from Wiz</h1></main></body>
</html>`

func handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = fmt.Fprint(w, page)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = fmt.Fprintln(w, "ok")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	server := &http.Server{Addr: ":" + port, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("listening on http://0.0.0.0:%s", port)
	log.Fatal(server.ListenAndServe())
}
