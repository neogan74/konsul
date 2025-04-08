package main

import (
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/kv/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			kvPut(w, r)
		case http.MethodGet:
			kvGet(w, r)
		case http.MethodDelete:
			kvDel(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	log.Println("Server started at http://localhost:8888")
	log.Fatal(http.ListenAndServe(":8888", nil))
}
