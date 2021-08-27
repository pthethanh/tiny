# tiny
Tiny helper for building tiny site with Go html/template quickly and easily.

Example: [example](https://github.com/pthethanh/tiny/tree/main/examples)
Demo: https://pthethanh.herokuapp.com/

```go
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
```
