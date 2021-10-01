# tiny
Tiny helper for building tiny site with Go html/template quickly and easily.

Example: [example](https://github.com/pthethanh/tiny/tree/main/examples)

One of the site built with `tiny`: https://pthethanh.herokuapp.com/

## Usage

### Template definition

- Use `[[]]` as template delimiter instead of `{{}}`.
- Data of each page will be available as below struct:
```
PageData struct {
	MetaData      MetaData
	Authenticated bool
	User          Claims
	Error         error
	Cookies       map[string]*http.Cookie

	// additional data return from DataHandler.
	Data interface{}
}
```
So, from template the page data can be accessed directly via the exposed properties:
```
[[.MetaData]]
[[.Authenticated]]
[[.User]]
[[.Error]]
[[.Cookies]]
[[.Data]]
```

### Starting the server

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

### Custom data handler

You can provide custom data handler by register it to the site using `SetDataHandler` method:

```
site := tiny.NewSite("index.yml")

// Set custom data handler.
site.SetDataHandler("index", func(rw http.ResponseWriter, r *http.Request) interface{} {
	return "hello"
})
```

Once set, the custom data can be accessed via `.Data` from the template:
```
<div>This is the data from custom data handler [[.Data]]</div>
```
