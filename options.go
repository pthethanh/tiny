package tiny

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/pthethanh/micro/status"
)

type (
	Option func(site *Site)
)

// Funcs add additional func map into the template engine.
func Funcs(funcs map[string]interface{}) Option {
	return func(site *Site) {
		if site.funcs == nil {
			site.funcs = make(map[string]interface{})
		}
		for k, v := range funcs {
			site.funcs[k] = v
		}
	}
}

func AuthInfo(f AuthInfoFunc) Option {
	return func(site *Site) {
		site.extractAuthInfo = f
	}
}

// JSONFileDataHandler return DataHandler that read data from the given file.
// Panics if failed to read the file.
func JSONFileDataHandler(f string, reload bool) DataHandler {
	loadData := func() (map[string]interface{}, error) {
		data := make(map[string]interface{})
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, status.Internal("read data from file, err: %v", err)
		}
		if err := json.Unmarshal(b, &data); err != nil {
			return nil, status.Internal("invalid data, err: %v", err)
		}
		return data, nil
	}
	data, err := loadData()
	if err != nil {
		panic(err)
	}
	return func(rw http.ResponseWriter, r *http.Request) interface{} {
		if !reload {
			return data
		}
		data, err := loadData()
		if err != nil {
			return err
		}
		return data
	}
}
