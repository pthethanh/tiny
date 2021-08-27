package main

import (
	"net/http"

	"github.com/pthethanh/tiny"
)

func main() {
	if err := http.ListenAndServe(":8000", tiny.NewSite("index.yml")); err != nil {
		panic(err)
	}
}
