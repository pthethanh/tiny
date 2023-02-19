package tiny

import (
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"time"
)

type (
	StaticSite struct {
		Enable       bool          `yaml:"enable"`
		Output       StaticOutput  `yaml:"output"`
		Static       []string      `yaml:"static"`
		AllowedPages []string      `yaml:"allowed_pages"`
		Request      StaticRequest `yaml:"request"`
	}

	StaticOutput struct {
		RootDir   string   `yaml:"root_dir"`
		StaticDir string   `yaml:"static_dir"`
		Keep      []string `yaml:"keep"`
	}

	StaticRequest struct {
		Host                 string   `yaml:"host"`
		Paths                []string `yaml:"paths"`
		dynamicPathsHandlers []DynamicPathsHandler
	}

	ResponseWriter struct {
		http.ResponseWriter
		body []byte
	}

	DynamicPathsHandler = func() []string
)

func (w *ResponseWriter) Write(b []byte) (int, error) {
	w.body = append(w.body, b...)
	return w.ResponseWriter.Write(b)
}

func (site *Site) AddDynamicPathsHandlers(hs ...DynamicPathsHandler) {
	site.StaticSite.Request.dynamicPathsHandlers = append(site.StaticSite.Request.dynamicPathsHandlers, hs...)
}

func (site *Site) prepareStaticSite() error {
	files, err := os.ReadDir(site.StaticSite.Output.RootDir)
	if err != nil {
		return err
	}
	keeps := map[string]bool{}
	for _, k := range site.StaticSite.Output.Keep {
		keeps[k] = true
	}
	// clean up old files
	for _, f := range files {
		if keeps[f.Name()] {
			continue
		}
		pth := filepath.Join(site.StaticSite.Output.RootDir, f.Name())
		if err := os.RemoveAll(pth); err != nil {
			return err
		}
	}
	// copy new static files
	for _, f := range site.StaticSite.Static {
		ff, err := os.Stat(f)
		if err != nil {
			return err
		}
		if ff.IsDir() {
			if err := copyDir(f, site.StaticSite.Output.StaticDir); err != nil {
				return err
			}
		} else {
			if err := copyFile(f, filepath.Join(site.StaticSite.Output.StaticDir, filepath.Base(f))); err != nil {
				return err
			}
		}
	}
	return nil
}

func (site *Site) staticGeneratorHandler() func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mw := &ResponseWriter{
				ResponseWriter: w,
			}
			h.ServeHTTP(mw, r)
			for _, page := range site.StaticSite.AllowedPages {
				if ok, _ := regexp.MatchString(page, r.URL.Path); ok {
					dir := path.Dir(r.URL.Path)
					name := path.Base(r.URL.Path)
					sep := "/"
					if dir == sep {
						if name == sep || name == "" {
							dir = ""
							name = "index.html"
						} else {
							dir = ""
							if filepath.Ext(name) == "" {
								name = name + ".html"
							}
						}
					} else {
						if name == sep || name == "" {
							dir = r.URL.Path
							name = "index.html"
						} else {
							if filepath.Ext(name) == "" {
								name = name + ".html"
							}
						}
					}
					dir = filepath.Join(site.StaticSite.Output.RootDir, dir)
					pth := filepath.Join(dir, name)
					if err := os.MkdirAll(dir, os.ModePerm); err != nil {
						log.Printf("error: %v", err)
						return
					}
					if dir == "" {
						pth = name
					}
					f, err := os.Create(pth)
					if err != nil {
						log.Printf("error: %v", err)
						return
					}
					defer f.Close()
					if _, err := f.Write(mw.body); err != nil {
						log.Printf("error: write static file failed, err: %v", err)
					}
				}
			}
		})
	}
}

func (site *Site) GenerateStaticSite() error {
	if !site.StaticSite.Enable {
		log.Println("warning: static site is disabled")
		return nil
	}
	if err := site.prepareStaticSite(); err != nil {
		log.Printf("error: failed to prepare static site, err: %v", err)
	}
	paths := site.StaticSite.Request.Paths
	for _, h := range site.StaticSite.Request.dynamicPathsHandlers {
		paths = append(paths, h()...)
	}
	c := http.Client{
		Timeout: 60 * time.Second,
	}
	defer c.CloseIdleConnections()
	for _, p := range paths {
		_, err := c.Get(site.StaticSite.Request.Host + p)
		if err != nil {
			return err
		}
	}
	return nil
}
