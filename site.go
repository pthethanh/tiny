package tiny

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/pthethanh/tiny/funcs"

	"gopkg.in/yaml.v3"
)

const (
	PageNotFound   = "404"
	PageError      = "500"
	PageRobotsTxt  = "robots.txt"
	PageSitemapXML = "sitemap.xml"
	filePrefix     = "file://"

	DefaultDelimLeft  = "[["
	DefaultDelimRight = "]]"
)

type (
	// Site hold site template config/definitions
	// this is just for quickly create a small site like blog.
	// Note that templates use tag [[ ]] by default.
	Site struct {
		MaxAge     time.Duration       `yaml:"max_age"`
		MetaData   MetaData            `yaml:"metadata"`
		Reload     bool                `yaml:"reload"`
		Login      string              `yaml:"login"`
		Layouts    map[string][]string `yaml:"layouts"`
		Pages      map[string]Page     `yaml:"pages"`
		Errors     map[string][]int    `yaml:"errors"`
		DelimLeft  string              `yaml:"delim_left"`
		DelimRight string              `yaml:"delim_right"`
		StaticSite StaticSite          `yaml:"static_site"`

		router    *mux.Router
		templates map[string]*template.Template
		mu        sync.RWMutex
		funcs     map[string]interface{}
		authInfo  AuthInfoFunc
		errors    map[int]string
	}

	// Page represent a web page.
	Page struct {
		Path        string        `yaml:"path"`
		Layout      string        `yaml:"layout"`
		Components  []string      `yaml:"components"`
		MetaData    MetaData      `yaml:"metadata"`
		Auth        bool          `yaml:"auth"`
		DelimLeft   string        `yaml:"delim_left"`
		DelimRight  string        `yaml:"delim_right"`
		Data        interface{}   `yaml:"data"`
		DataType    string        `yaml:"data_type"`
		MaxAge      time.Duration `yaml:"max_age"`
		DataHandler DataHandler   `yaml:"-"`

		isStatic bool
	}
	// PageData hold basic data of a web page.
	PageData struct {
		MetaData      MetaData
		Authenticated bool
		User          interface{}
		Error         error
		Cookies       map[string]*http.Cookie

		// additional data return from DataHandler.
		Data interface{}
	}
	SiteMapURL struct {
		Loc        string
		LastMod    time.Time
		ChangeFreq string
		Priority   float64
	}
	SiteMap struct {
		URLSet []SiteMapURL
	}

	UserAgent struct {
		UserAgent string
		Disallow  []string
		Allow     []string
	}
	RobotsTXT struct {
		UserAgents []UserAgent
	}

	// DataHandler is a custom handler for providing data to be used in page templates.
	// The return data can be used in template via `.Data` property of PageData.
	// If the return data is a PageData, the default PageData will be overridden.
	DataHandler          = func(rw http.ResponseWriter, r *http.Request) interface{}
	SiteMapDataHandler   = func(rw http.ResponseWriter, r *http.Request) SiteMap
	RobotsTXTDataHandler = func(rw http.ResponseWriter, r *http.Request) RobotsTXT
	AuthInfoFunc         = func(context.Context) (interface{}, bool)
)

// NewSite read site definition from yaml config file.
// Panics if any error.
func NewSite(path string, options ...Option) *Site {
	b, err := os.ReadFile(path)
	if err != nil {
		log.Panic(err)
	}
	site := Site{
		MetaData: map[string]interface{}{
			"lang":          "en",
			"author":        "tiny",
			"description":   "Tiny",
			"domain":        "localhost",
			"key_words":     []string{"tiny"},
			"title":         "Tiny",
			"type":          "WebSite",
			"site_name":     "Tiny",
			"version":       "v0.0.1",
			"image":         "",
			"base_url":      "",
			"canonical_url": "",
		},
		Pages: map[string]Page{},
		Errors: map[string][]int{
			PageNotFound: {http.StatusNotFound},
			PageError:    {http.StatusInternalServerError},
		},
		errors:     make(map[int]string),
		mu:         sync.RWMutex{},
		funcs:      funcs.FuncMap(),
		templates:  make(map[string]*template.Template),
		DelimLeft:  DefaultDelimLeft,
		DelimRight: DefaultDelimRight,
		MaxAge:     30 * 24 * time.Hour,
	}
	// parse config
	if err := yaml.Unmarshal(b, &site); err != nil {
		log.Panic(err)
	}
	// apply user options
	for _, opt := range options {
		opt(&site)
	}
	// re-mapping error handlers
	for p, errs := range site.Errors {
		for _, err := range errs {
			site.errors[err] = p
		}
	}
	// setup handlers and routers
	site.setupDataHandlers()
	site.setupRouter()

	// validate site config
	if err := site.validateSite(); err != nil {
		log.Panic(err)
	}
	return &site
}

func (site *Site) setupDataHandlers() {
	for n, p := range site.Pages {
		n := n
		p := p
		if p.MaxAge <= 0 {
			p.MaxAge = site.MaxAge
		}
		switch {
		case p.DataType == "json":
			f, ok := p.Data.(string)
			if !ok {
				log.Panicf("invalid data type, page: %s, data: %v", n, p.Data)
			}
			f = f[len(filePrefix):]
			site.SetDataHandler(n, site.jsonFileDataHandler(f))
		case strings.HasPrefix(fmt.Sprintf("%v", p.Data), filePrefix):
			f, ok := p.Data.(string)
			if !ok {
				log.Panicf("invalid data type, page: %s, data: %v", n, p.Data)
			}
			f = f[len(filePrefix):]
			site.SetDataHandler(n, site.fileDataHandler(p.Path, f, p.MaxAge))
			pp := site.Pages[n]
			pp.isStatic = true
			site.Pages[n] = pp
		default:
			// serve it as raw data
			site.SetDataHandler(n, func(rw http.ResponseWriter, r *http.Request) interface{} {
				return p.Data
			})
		}
	}
}

func (site *Site) setupRouter() {
	router := mux.NewRouter()
	for name, p := range site.Pages {
		log.Printf("info: register page: %s, path: %s, method: %s\n", name, p.Path, http.MethodGet)
		h := site.getPageHandler(name)
		if p.Auth {
			h = AuthRequired(site.Login, site.authInfo)(h)
		}
		if p.isStaticDir() {
			router.PathPrefix(p.Path).Methods(http.MethodGet).Handler(h)
		} else {
			router.Path(p.Path).Methods(http.MethodGet).Handler(h)
		}
	}
	router.NotFoundHandler = site.getPageHandler(PageNotFound)
	site.router = router
}

// getPageData get common data from configuration and request.
func (site *Site) getPageData(pageName string, rw http.ResponseWriter, r *http.Request) PageData {
	// get claims information.
	var claims interface{}
	authenticated := false
	if site.authInfo != nil {
		claims, authenticated = site.authInfo(r.Context())
	}
	// get metadata
	data := PageData{
		MetaData:      site.getPageMetaData(pageName),
		Authenticated: authenticated,
		User:          claims,
		Error:         nil,
		Cookies:       make(map[string]*http.Cookie),
	}
	// collect cookies if any.
	for _, ck := range r.Cookies() {
		data.Cookies[ck.Name] = ck
	}
	// get data from data handler if any
	p, ok := site.Pages[pageName]
	if ok && p.DataHandler != nil {
		d := p.DataHandler(rw, r)
		if pd, ok := d.(PageData); ok {
			// new data takes priority.
			data.merge(pd)
		} else if err, ok := d.(error); ok {
			data.Error = err
		} else {
			data.Data = d
		}
	} else {
		// in case we have predefined data.
		data.Data = p.Data
	}
	return data
}

func (site *Site) getPageHandler(name string) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if _, ok := site.Pages[name]; !ok {
			http.Error(rw, "Page Not Found", http.StatusNotFound)
			return
		}
		data := site.getPageData(name, rw, r)
		if data.Error != nil {
			site.handleError(rw, r, data.Error)
			return
		}
		if site.Pages[name].isStatic {
			return
		}
		data.MetaData.SetCanonicalURL(data.MetaData.BaseURL() + r.URL.Path)
		if err := site.handlePage(rw, r, name, data); err != nil {
			log.Printf("error: template:%s, err: %v\n", name, err)
			site.handleError(rw, r, err)
			return
		}
	})
}

func (site *Site) fileDataHandler(prefix string, f string, maxAge time.Duration) DataHandler {
	return func(rw http.ResponseWriter, r *http.Request) interface{} {
		ff, err := os.Stat(f)
		if err != nil {
			return err
		}
		if ff.IsDir() {
			h := Cache(maxAge)(http.StripPrefix(prefix, http.FileServer(http.Dir(f))))
			h.ServeHTTP(rw, r)
			return nil
		}
		http.ServeFile(rw, r, f)
		return nil
	}
}

// ServeHTTP serve the configured pages.
func (site *Site) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	if site.StaticSite.Enable {
		site.staticGeneratorHandler()(site.router).ServeHTTP(rw, r)
		return
	}
	site.router.ServeHTTP(rw, r)
}

func (site *Site) SetDataHandlers(handlers map[string]DataHandler) error {
	for name, h := range handlers {
		if err := site.SetDataHandler(name, h); err != nil {
			return err
		}
	}
	return nil
}

func (site *Site) SetDataHandler(name string, h DataHandler) error {
	site.mu.Lock()
	defer site.mu.Unlock()
	p, ok := site.Pages[name]
	if !ok {
		return NewError(http.StatusNotFound, "page not found")
	}
	p.DataHandler = h
	site.Pages[name] = p
	return nil
}

func (site *Site) SetSiteMapDataHandler(name string, h SiteMapDataHandler) error {
	return site.SetDataHandler(name, func(rw http.ResponseWriter, r *http.Request) interface{} {
		return h(rw, r)
	})
}

func (site *Site) SetRobotsTXTDataHandler(name string, h RobotsTXTDataHandler) error {
	return site.SetDataHandler(name, func(rw http.ResponseWriter, r *http.Request) interface{} {
		return h(rw, r)
	})
}

// getPageMetaData get metadata from config.
func (site *Site) getPageMetaData(name string) MetaData {
	page, ok := site.Pages[name]
	if !ok {
		return site.MetaData
	}
	if page.MetaData == nil {
		page.MetaData = make(MetaData)
	}
	// get value from site if page doesn't defined.
	for k, v := range site.MetaData {
		if _, ok := page.MetaData[k]; !ok {
			page.MetaData[k] = v
		}
	}
	return page.MetaData
}

// parseTemplate parse the template base on the given config name.
func (site *Site) parseTemplate(name string) (*template.Template, error) {
	tpl, loaded := site.templates[name]
	// if loaded and Reload is disabled, return.
	if loaded && !site.Reload {
		return tpl, nil
	}
	// parse the template.
	page, ok := site.Pages[name]
	if !ok {
		return nil, NewError(http.StatusNotFound, "page not found")
	}
	layouts := site.Layouts[page.Layout]
	files := append(layouts, page.Components...)
	if len(files) == 0 {
		return nil, NewError(http.StatusNotFound, "no templates found")
	}
	tplName := page.Layout
	if page.Layout == "" {
		tplName = path.Base(files[0])
	}
	if path.Ext(tplName) == "" {
		tplName = fmt.Sprintf("%s.html", tplName)
	}
	// load predefined template with default delims.
	tpl = template.New(tplName).Delims(DefaultDelimLeft, DefaultDelimRight).Funcs(site.funcs)
	// delims can be overridden page by page.
	delimLeft, delimRight := page.DelimLeft, page.DelimRight
	if delimLeft == "" || delimRight == "" {
		delimLeft, delimRight = site.DelimLeft, site.DelimRight
	}
	tpl, err := tpl.Delims(delimLeft, delimRight).ParseFiles(files...)
	if err != nil {
		log.Printf("error: parse template, err: %v\n", err)
		return nil, err
	}
	site.templates[name] = tpl
	return tpl, nil
}

func (site *Site) handleError(rw http.ResponseWriter, r *http.Request, err error) {
	name := PageError
	if t, ok := site.errors[ErrorFromErr(err).Code()]; ok {
		name = t
	}
	if _, ok := site.Pages[name]; !ok {
		http.Error(rw, "Page Not Found", http.StatusNotFound)
		return
	}
	data := site.getPageData(name, rw, r)
	data.Error = err
	if err := site.handlePage(rw, r, name, data); err != nil {
		log.Printf("error: serve error page, err: %v\n", err)
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (site *Site) handlePage(w http.ResponseWriter, r *http.Request, name string, data interface{}) error {
	t, err := site.parseTemplate(name)
	if err != nil {
		log.Printf("error: %s parse failed, err: %v\n", name, err)
		return err
	}
	if data == nil {
		data = site.getPageData(name, w, r)
	}
	if err := t.Execute(w, data); err != nil {
		return err
	}
	return nil
}

func (page PageData) GetCookie(k string) string {
	if ck, ok := page.Cookies[k]; ok {
		return ck.Value
	}
	return ""
}

// jsonFileDataHandler return DataHandler that read data from the given json file.
// Data can be accessed via .Data in templates.
// Panics if failed to read the file.
func (site *Site) jsonFileDataHandler(f string) DataHandler {
	loadData := func() (interface{}, error) {
		var data interface{}
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, NewError(http.StatusInternalServerError, "read data from file, err: %v", err)
		}
		if err := json.Unmarshal(b, &data); err != nil {
			return nil, NewError(http.StatusInternalServerError, "invalid data, err: %v", err)
		}
		return data, nil
	}
	data, err := loadData()
	if err != nil {
		panic(err)
	}
	return func(rw http.ResponseWriter, r *http.Request) interface{} {
		if !site.Reload {
			return data
		}
		data, err := loadData()
		if err != nil {
			return err
		}
		return data
	}
}

func (site *Site) validateSite() error {
	for l, comps := range site.Layouts {
		for _, c := range comps {
			if _, err := os.Stat(c); err != nil {
				return fmt.Errorf("layout: %s, component: %s, err: %w", l, c, err)
			}
		}
	}
	auth := false
	// validate if configured files exists
	for n, p := range site.Pages {
		// check if layout exists
		if _, ok := site.Layouts[p.Layout]; !ok && p.Layout != "" {
			return fmt.Errorf("page:%s, layout: %s not found", n, p.Layout)
		}
		// check if component exists
		for _, c := range p.Components {
			if _, err := os.Stat(c); err != nil {
				return fmt.Errorf("page: %s, component: %s, err: %w", n, c, err)
			}
		}
		auth = auth || p.Auth
	}
	if auth && site.authInfo == nil {
		return fmt.Errorf("auth is enabled but no auth info func is provided")
	}
	return nil
}

// mergeFrom merge the page1 data to the current page data
func (page *PageData) merge(page1 PageData) {
	page.Data = page1.Data
	page.Error = page1.Error
	for k, v := range page1.MetaData {
		page.MetaData[k] = v
	}
	for k, ck := range page1.Cookies {
		page.Cookies[k] = ck
	}
	if page1.User != nil {
		page.Authenticated = page1.Authenticated
		page.User = page1.User
	}
}

func (p Page) isStaticDir() bool {
	if !p.isStatic {
		return false
	}
	f := p.Data.(string)[len(filePrefix):]
	ff, err := os.Stat(f)
	if err != nil {
		return false
	}
	return ff.IsDir()
}
