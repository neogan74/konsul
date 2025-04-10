package main

import (
	"log"
	"net/http"

	"github.com/neogan74/konsul/internal/api"
)

func main() {
	http.HandleFunc("/kv/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			api.KvPut(w, r)
		case http.MethodGet:
			api.KvGet(w, r)
		case http.MethodDelete:
			api.KvDel(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	log.Println("Server started at http://localhost:8888")
	log.Fatal(http.ListenAndServe(":8888", nil))
}
