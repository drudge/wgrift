package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	dir := "ui/web"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		dir = "."
	}

	addr := ":8080"
	fmt.Printf("Serving %s at http://localhost%s\n", dir, addr)
	fs := http.FileServer(http.Dir(dir))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		fs.ServeHTTP(w, r)
	})
	log.Fatal(http.ListenAndServe(addr, handler))
}
