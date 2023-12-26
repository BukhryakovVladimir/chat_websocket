package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"text/template"

	"github.com/rs/cors"
)

// templ represents a single template
type templateHandler struct {
	once     sync.Once
	filename string
	templ    *template.Template
}

// ServeHTTP handles the HTTP request.
func (t *templateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.once.Do(func() {
		t.templ = template.Must(template.ParseFiles(filepath.Join("templates", t.filename)))
	})
	t.templ.Execute(w, r)
}

func readCache(w http.ResponseWriter, r *http.Request) {
	file, err := os.Open("cache.txt")

	defer file.Close()

	messages, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("Error:", err)
	}

	w.WriteHeader(http.StatusOK)
	w.Write(messages)
}

func main() {
	var addr = flag.String("addr", ":8080", "The addr of the application.")
	flag.Parse() // parse the flags

	r := newRoom()

	mux := http.NewServeMux()
	mux.HandleFunc("/readCache", readCache)
	mux.Handle("/", &templateHandler{filename: "chat.html"})
	mux.Handle("/room", r)

	handler := cors.Default().Handler(mux)
	// get the room going
	go r.run()

	// start the web server
	log.Println("Starting web server on", *addr)
	if err := http.ListenAndServe(*addr, handler); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
