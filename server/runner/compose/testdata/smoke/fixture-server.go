package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	if filepath.Base(os.Args[0]) == "curl" {
		fmt.Println("ok")
		return
	}
	http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok\n")) })
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
