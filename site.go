package tiny

import (
	"embed"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"path"
	"reflect"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/pthethanh/micro/auth/jwt"
	"github.com/pthethanh/micro/log"
	"github.com/pthethanh/micro/status"
	"github.com/pthethanh/tiny/funcs"
	"google.golang.org/grpc/codes"

	"gopkg.in/yaml.v3"
)

var (
	//go:embed templates
	defaultTemplates embed.FS
)

const (
	pageNotFound = "not_found"
	pageError    = "error"
)

type (
	// Site hold site template config/definitions
	// this is just for quickly create a small site like blog.
	// Note that templates must use tag [[.]] instead of {{.}}
	Site struct {
		CacheMaxAge   time.Duration       `yaml:"cache_max_age"`
		MetaData      MetaData            `yaml:"metadata"`
		Reload        bool                `yaml:"reload"`
		Static        string              `yaml:"static"`
		StaticPrefix  string              `yaml:"static_prefix"`
		Layouts       map[string][]string `yaml:"layouts"`
		Pages         map[string]Page     `yaml:"pages"`
		ErrorHandlers map[uint32]string   `yaml:"error_handlers"`
		Login         string              `yaml:"login"`

		once      sync.Once
		router    *mux.Router
		templates map[string]*template.Template
		mu        sync.RWMutex
		funcs     map[string]interface{}
	}

	DataHandler          func(rw http.ResponseWriter, r *http.Request) interface{}
	SiteMapDataHandler   func(rw http.ResponseWriter, r *http.Request) SiteMap
	RobotsTXTDataHandler func(rw http.ResponseWriter, r *http.Request) RobotsTXT

	DataHandlerService interface {
		DataHandlers() map[string]DataHandler
	}

	// Page prepresent a web page.
	Page struct {
		Path        string   `yaml:"path"`
		Layout      string   `yaml:"layout"`
		Components  []string `yaml:"components"`
		MetaData    MetaData `yaml:"metadata"`
		Auth        bool     `yaml:"auth"`
		DataHandler DataHandler

		embed bool
	}
	// PageData hold basic data of a web page.
	PageData struct {
		MetaData      MetaData
		Authenticated bool
		User          jwt.Claims
		Error         status.Code
		Cookies       map[string]*http.Cookie
	}

	// MetaData hold metadata of a page.
	MetaData struct {
		Version      string   `yaml:"version"`
		Lang         string   `yaml:"lang"`
		SiteName     string   `yaml:"site_name"`
		Title        string   `yaml:"title"`
		Domain       string   `yaml:"domain"`
		BaseURL      string   `yaml:"base_url"`
		CanonicalURL string   `yaml:"canonical_url"`
		KeyWords     []string `yaml:"key_words"`
		Author       string   `yaml:"author"`
		Type         string   `yaml:"type"`
		Image        string   `yaml:"image"`
		Description  string   `yaml:"description"`
	}
	SiteMapURL struct {
		Loc        string
		LastMod    time.Time
		ChangeFreq string
		Priority   float64
	}
	SiteMap struct {
		PageData
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
)

// NewSite read site definition from yaml config file.
// Panics if any error.
func NewSite(path string, options ...Option) *Site {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		log.Panic(err)
	}
	site := Site{
		Static: "web/dist/static/",
		MetaData: MetaData{
			Lang:        "en",
			Author:      "tiny",
			Description: "Tiny",
			Domain:      "localhost",
			KeyWords:    []string{"tiny"},
			Title:       "Tiny",
			Type:        "WebSite",
			SiteName:    "Tiny",
		},
		Pages: map[string]Page{},
		mu:    sync.RWMutex{},
		ErrorHandlers: map[uint32]string{
			uint32(codes.NotFound): pageNotFound,
		},
	}
	if err := yaml.Unmarshal(b, &site); err != nil {
		log.Panic(err)
	}
	options = append(options, Funcs(funcs.FuncMap()))
	for _, opt := range options {
		opt(&site)
	}
	site.templates = make(map[string]*template.Template)
	site.addDefaultPagesIfNotExists()
	if site.MetaData.Version == "" {
		site.MetaData.Version = "v0.0.1"
	}
	if site.funcs == nil {
		site.funcs = funcs.FuncMap()
	}
	if site.StaticPrefix == "" {
		site.StaticPrefix = "/static/"
	}
	return &site
}

// GetPageData get common data from configuration and request.
func (site *Site) GetPageData(pageName string, r *http.Request, err ...error) PageData {
	authenticated := false
	claims, authenticated := jwt.FromContext(r.Context())
	code := status.OK("").Code()
	if len(err) > 0 {
		code = status.Convert(err[0]).Code()
	}
	data := PageData{
		MetaData:      site.getPageMetaData(pageName),
		Authenticated: authenticated,
		User:          claims,
		Error:         code,
		Cookies:       make(map[string]*http.Cookie),
	}
	for _, ck := range r.Cookies() {
		data.Cookies[ck.Name] = ck
	}
	return data
}

func (site *Site) ServePage(name string) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		var data interface{}
		if p, ok := site.Pages[name]; ok && p.DataHandler != nil {
			data = p.DataHandler(rw, r)
		} else {
			data = site.GetPageData(name, r, nil)
		}
		if err, ok := data.(error); ok {
			site.handleError(rw, r, err)
			return
		}
		if err := site.handlePage(rw, r, name, data); err != nil {
			log.Context(r.Context()).Errorf("template:%s, err: %v", name, err)
			site.handleError(rw, r, err)
			return
		}
	})
}

func (site *Site) ServeStatic(prefix string) http.Handler {
	return Cache(int64(site.CacheMaxAge.Seconds()))(http.StripPrefix(prefix, http.FileServer(http.Dir(site.Static))))
}

// ServeHTTP serve the configured pages.
func (site *Site) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	site.once.Do(func() {
		router := mux.NewRouter()
		router.PathPrefix(site.StaticPrefix).Handler(site.ServeStatic(site.StaticPrefix))
		for name, p := range site.Pages {
			log.Infof("register page: %s, path: %s, method: %s", name, p.Path, http.MethodGet)
			h := site.ServePage(name)
			if p.Auth {
				h = AuthRequired(site.Login)(h)
			}
			router.Path(p.Path).Methods(http.MethodGet).Handler(h)
		}
		router.NotFoundHandler = site.ServePage(pageNotFound)
		site.router = router
	})
	site.router.ServeHTTP(rw, r)
}

func (site *Site) RegisterDataHandlers(services ...DataHandlerService) {
	for _, srv := range services {
		site.SetDataHandlers(srv.DataHandlers())
	}
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
		return status.NotFound("page not found")
	}
	p.DataHandler = h
	site.Pages[name] = p
	return nil
}

func (site *Site) SetSiteMapDataHandler(name string, h SiteMapDataHandler) error {
	return site.SetDataHandler(name, func(rw http.ResponseWriter, r *http.Request) interface{} {
		data := h(rw, r)
		if reflect.DeepEqual(data.PageData, PageData{}) {
			data.PageData = site.GetPageData(name, r)
		}
		return data
	})
}

func (site *Site) SetRobotsTXTDataHandler(name string, h RobotsTXTDataHandler) error {
	return site.SetDataHandler(name, func(rw http.ResponseWriter, r *http.Request) interface{} {
		return h(rw, r)
	})
}

func (site *Site) addDefaultPagesIfNotExists() {
	if _, ok := site.Pages["robots.txt"]; !ok {
		site.addDefaultRobotsTxt()
	}
	if _, ok := site.Pages["sitemap.xml"]; !ok {
		site.addDefaultSiteMap()
	}
	if _, ok := site.Pages[pageError]; !ok {
		site.addEmbedPage(pageError, defaultTemplates, "templates/error.html", Page{
			Path: "/error",
		})
	}
	if _, ok := site.Pages[pageNotFound]; !ok {
		site.addEmbedPage(pageNotFound, defaultTemplates, "templates/not_found.html", Page{
			Path: "/404",
		})
	}
}

// getPageMetaData get metadata from config.
func (site *Site) getPageMetaData(name string) MetaData {
	page, ok := site.Pages[name]
	if !ok {
		return site.MetaData
	}
	meta := page.MetaData
	// general information should not be overridden
	meta.Domain = site.MetaData.Domain
	meta.SiteName = site.MetaData.SiteName
	meta.BaseURL = site.MetaData.BaseURL

	// specific info can be overridden.
	if meta.Image == "" {
		meta.Image = site.MetaData.Image
	}
	if meta.Title == "" {
		meta.Title = site.MetaData.Title
	}
	if meta.Author == "" {
		meta.Author = site.MetaData.Author
	}
	if len(meta.KeyWords) == 0 {
		meta.KeyWords = append(meta.KeyWords, site.MetaData.KeyWords...)
	}
	if meta.Description == "" {
		meta.Description = site.MetaData.Description
	}
	if meta.Lang == "" {
		meta.Lang = site.MetaData.Lang
	}
	if meta.CanonicalURL == "" {
		meta.CanonicalURL = site.MetaData.CanonicalURL
	}
	if meta.Author == "" {
		meta.Author = site.MetaData.Author
	}
	if meta.Type == "" {
		meta.Type = site.MetaData.Type
	}
	if meta.Image == "" {
		meta.Image = site.MetaData.Image
	}
	if meta.Version == "" {
		meta.Version = site.MetaData.Version
	}
	return meta
}

// parseTemplate parse the template base on the given config name.
// use [[]] for delimiter tag.
func (site *Site) parseTemplate(name string) (*template.Template, error) {
	tpl, loaded := site.templates[name]
	if loaded && site.Pages[name].embed {
		return tpl, nil
	}
	if !loaded || site.Reload {
		page, ok := site.Pages[name]
		if !ok {
			return nil, status.NotFound("page not found")
		}
		layout := site.Layouts[page.Layout]
		files := append(layout, page.Components...)
		if len(files) == 0 {
			return nil, status.NotFound("no templates found")
		}
		tplName := page.Layout
		if page.Layout == "" {
			tplName = path.Base(files[0])
		}
		if path.Ext(tplName) == "" {
			tplName = fmt.Sprintf("%s.html", tplName)
		}
		t, err := template.New(tplName).Delims("[[", "]]").Funcs(site.funcs).ParseFS(defaultTemplates, "templates/common.html")
		if err != nil {
			log.Errorf("template: parse common template, err: %v", err)
			return nil, err
		}
		t, err = t.ParseFiles(files...)
		if err != nil {
			log.Errorf("template: parse template, err: %v", err)
			return nil, err
		}
		site.templates[name] = t
		tpl = t
	}
	return tpl, nil
}

func (site *Site) handleError(rw http.ResponseWriter, r *http.Request, err error) {
	name := pageError
	if t, ok := site.ErrorHandlers[uint32(status.Convert(err).Code())]; ok {
		name = t
	}
	if err := site.handlePage(rw, r, name, site.GetPageData(name, r, err)); err != nil {
		log.Context(r.Context()).Errorf("template: serve error page, err: %v", err)
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (site *Site) handlePage(w http.ResponseWriter, r *http.Request, name string, data interface{}) error {
	t, err := site.parseTemplate(name)
	if err != nil {
		log.Context(r.Context()).Errorf("template:%s parse failed, err: %v", name, err)
		return err
	}
	if data == nil {
		data = site.GetPageData(name, r)
	}
	if err := t.Execute(w, data); err != nil {
		return err
	}
	return nil
}

func (site *Site) addEmbedPage(name string, fs embed.FS, pattern string, p Page) {
	t, err := template.New(path.Base(pattern)).Delims("[[", "]]").Funcs(site.funcs).ParseFS(fs, pattern)
	if err != nil {
		log.Panic(err)
	}
	p.embed = true

	site.templates[name] = t
	site.Pages[name] = p
}

func (site *Site) addDefaultSiteMap() {
	site.addEmbedPage("sitemap.xml", defaultTemplates, "templates/sitemap.xml", Page{
		Path: "/sitemap.xml",
		DataHandler: func(rw http.ResponseWriter, r *http.Request) interface{} {
			return SiteMap{
				PageData: site.GetPageData("sitemap.xml", r),
				URLSet: []SiteMapURL{
					{
						Loc:        "/",
						LastMod:    time.Now(),
						ChangeFreq: "daily",
						Priority:   1,
					},
				},
			}
		},
	})
}

func (site *Site) addDefaultRobotsTxt() {
	site.addEmbedPage("robots.txt", defaultTemplates, "templates/robots.txt", Page{
		Path: "/robots.txt",
		DataHandler: func(rw http.ResponseWriter, r *http.Request) interface{} {
			return RobotsTXT{
				UserAgents: []UserAgent{
					{
						UserAgent: "*",
						Allow:     []string{"/"},
					},
				},
			}
		},
	})
}

func (page PageData) GetCookie(k string) string {
	if ck, ok := page.Cookies[k]; ok {
		return ck.Value
	}
	return ""
}
