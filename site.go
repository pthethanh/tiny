package tiny

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/pthethanh/tiny/funcs"

	"gopkg.in/yaml.v3"
)

var (
	//go:embed templates
	defaultTemplates embed.FS
)

const (
	PageNotFound   = "not_found"
	PageError      = "error"
	PageRobotsTxt  = "robots.txt"
	PageSitemapXML = "sitemap.xml"

	DefaultDelimeLeft  = "[["
	DefaultDelimeRight = "]]"
)

type (
	// Site hold site template config/definitions
	// this is just for quickly create a small site like blog.
	// Note that templates use tag [[ ]] by default.
	Site struct {
		CacheMaxAge  time.Duration       `yaml:"cache_max_age"`
		MetaData     MetaData            `yaml:"metadata"`
		Reload       bool                `yaml:"reload"`
		Static       string              `yaml:"static"`
		StaticPrefix string              `yaml:"static_prefix"`
		Login        string              `yaml:"login"`
		Layouts      map[string][]string `yaml:"layouts"`
		Pages        map[string]Page     `yaml:"pages"`
		Errors       map[uint32]string   `yaml:"errors"`
		Validate     bool                `yaml:"validate"`
		DelimLeft    string              `yaml:"delim_left"`
		DelimRight   string              `yaml:"delim_right"`

		once      sync.Once
		router    *mux.Router
		templates map[string]*template.Template
		mu        sync.RWMutex
		funcs     map[string]interface{}
		authInfo  AuthInfoFunc
	}

	// Claims represents the claims provided by the JWT.
	Claims struct {
		// Auth claims
		Audience  string `json:"aud,omitempty"`
		ExpiresAt int64  `json:"exp,omitempty"`
		ID        string `json:"jti,omitempty"`
		IssuedAt  int64  `json:"iat,omitempty"`
		Issuer    string `json:"iss,omitempty"`
		NotBefore int64  `json:"nbf,omitempty"`
		Subject   string `json:"sub,omitempty"`

		// User attributes claims
		Name                string `json:"name,omitempty"`
		GivenName           string `json:"given_name,omitempty"`
		FamilyName          string `json:"family_name,omitempty"`
		MiddleName          string `json:"middle_name,omitempty"`
		Nickname            string `json:"nickname,omitempty"`
		PreferredUsername   string `json:"preferred_username,omitempty"`
		Profile             string `json:"profile,omitempty"`
		Picture             string `json:"picture,omitempty"`
		Website             string `json:"website,omitempty"`
		Email               string `json:"email,omitempty"`
		EmailVerified       bool   `json:"email_verified,omitempty"`
		Gender              string `json:"gender,omitempty"`
		BirthDate           string `json:"birthdate,omitempty"`
		ZoneInfo            string `json:"zoneinfo,omitempty"`
		Locale              string `json:"locale,omitempty"`
		PhoneNumber         string `json:"phone_number,omitempty"`
		PhoneNumberVerified bool   `json:"phone_number_verified,omitempty"`
		Address             string `json:"address,omitempty"`
		UpdatedAt           int64  `json:"updated_at,omitempty"`

		// Custom attributes claims.
		Scope    string                 `json:"scope,omitempty"`
		Admin    bool                   `json:"admin,omitempty"`
		Metadata map[string]interface{} `json:"metadata,omitempty"`
	}

	// Page represent a web page.
	Page struct {
		Path        string   `yaml:"path"`
		Layout      string   `yaml:"layout"`
		Components  []string `yaml:"components"`
		MetaData    MetaData `yaml:"metadata"`
		Auth        bool     `yaml:"auth"`
		Data        string   `yaml:"data"`
		DelimLeft   string   `yaml:"delim_left"`
		DelimRight  string   `yaml:"delim_right"`
		DataHandler DataHandler

		embed bool
	}
	// PageData hold basic data of a web page.
	PageData struct {
		MetaData      MetaData
		Authenticated bool
		User          Claims
		Error         error
		Cookies       map[string]*http.Cookie

		// additional data return from DataHandler.
		Data interface{}
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
	AuthInfoFunc         = func(context.Context) (Claims, bool)
)

// NewSite read site definition from yaml config file.
// Panics if any error.
func NewSite(path string, options ...Option) *Site {
	b, err := os.ReadFile(path)
	if err != nil {
		log.Panic(err)
	}
	site := Site{
		MetaData: MetaData{
			Lang:        "en",
			Author:      "tiny",
			Description: "Tiny",
			Domain:      "localhost",
			KeyWords:    []string{"tiny"},
			Title:       "Tiny",
			Type:        "WebSite",
			SiteName:    "Tiny",
			Version:     "v0.0.1",
		},
		Pages: map[string]Page{},
		Errors: map[uint32]string{
			http.StatusNotFound: PageNotFound,
		},
		mu:         sync.RWMutex{},
		funcs:      funcs.FuncMap(),
		templates:  make(map[string]*template.Template),
		Validate:   true,
		DelimLeft:  DefaultDelimeLeft,
		DelimRight: DefaultDelimeRight,
	}
	if err := yaml.Unmarshal(b, &site); err != nil {
		log.Panic(err)
	}
	// apply user options
	for _, opt := range options {
		opt(&site)
	}
	// add default pages if not exists
	site.addDefaultPagesIfNotExists()
	// set data handler from JSON file if defined.
	for n, p := range site.Pages {
		if p.Data != "" {
			site.SetDataHandler(n, site.jsonFileDataHandler(p.Data))
		}
	}
	// validate config file.
	if !site.Validate {
		log.Printf("warn: config validation is disabled.")
	}
	if err := site.validateSite(); err != nil {
		if site.Validate {
			log.Panic(err)
		}
		log.Printf("warn: invalid config file: %v\n", err)
	}
	return &site
}

// GetPageData get common data from configuration and request.
func (site *Site) GetPageData(pageName string, r *http.Request, errs ...error) PageData {
	claims := Claims{}
	authenticated := false
	if site.authInfo != nil {
		claims, authenticated = site.authInfo(r.Context())
	}
	var err error
	if len(errs) > 0 {
		err = errs[0]
	}
	data := PageData{
		MetaData:      site.getPageMetaData(pageName),
		Authenticated: authenticated,
		User:          claims,
		Error:         err,
		Cookies:       make(map[string]*http.Cookie),
	}
	for _, ck := range r.Cookies() {
		data.Cookies[ck.Name] = ck
	}
	return data
}

func (site *Site) ServePage(name string) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		data := site.GetPageData(name, r, nil)
		if p, ok := site.Pages[name]; ok && p.DataHandler != nil {
			d := p.DataHandler(rw, r)
			if pd, ok := d.(PageData); ok {
				data = pd
			} else {
				data.Data = d
			}
		}
		if err, ok := data.Data.(error); ok {
			site.handleError(rw, r, err)
			return
		}
		if err := site.handlePage(rw, r, name, data); err != nil {
			log.Printf("error: template:%s, err: %v\n", name, err)
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
			log.Printf("info: register page: %s, path: %s, method: %s\n", name, p.Path, http.MethodGet)
			h := site.ServePage(name)
			if p.Auth {
				h = AuthRequired(site.Login, site.authInfo)(h)
			}
			router.Path(p.Path).Methods(http.MethodGet).Handler(h)
		}
		router.NotFoundHandler = site.ServePage(PageNotFound)
		site.router = router
	})
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
		return Error(http.StatusNotFound, "page not found")
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

func (site *Site) addDefaultPagesIfNotExists() {
	if _, ok := site.Pages[PageRobotsTxt]; !ok {
		site.addDefaultRobotsTxt()
	}
	if _, ok := site.Pages[PageSitemapXML]; !ok {
		site.addDefaultSiteMap()
	}
	if _, ok := site.Pages[PageError]; !ok {
		site.addEmbedPage(PageError, defaultTemplates, "templates/error.html", Page{
			Path: "/error",
		})
	}
	if _, ok := site.Pages[PageNotFound]; !ok {
		site.addEmbedPage(PageNotFound, defaultTemplates, "templates/not_found.html", Page{
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
func (site *Site) parseTemplate(name string) (*template.Template, error) {
	tpl, loaded := site.templates[name]
	// if it's embed template, no need  to parse again.
	if loaded && site.Pages[name].embed {
		return tpl, nil
	}
	// if loaded and Reload is disabled, return.
	if loaded && !site.Reload {
		return tpl, nil
	}
	// parse the template.
	page, ok := site.Pages[name]
	if !ok {
		return nil, Error(http.StatusNotFound, "page not found")
	}
	layout := site.Layouts[page.Layout]
	files := append(layout, page.Components...)
	if len(files) == 0 {
		return nil, Error(http.StatusNotFound, "no templates found")
	}
	tplName := page.Layout
	if page.Layout == "" {
		tplName = path.Base(files[0])
	}
	if path.Ext(tplName) == "" {
		tplName = fmt.Sprintf("%s.html", tplName)
	}
	// load predefined template with default delims.
	tpl, err := template.New(tplName).Delims(DefaultDelimeLeft, DefaultDelimeRight).Funcs(site.funcs).ParseFS(defaultTemplates, "templates/common.html")
	if err != nil {
		log.Printf("error: parse common template, err: %v\n", err)
		return nil, err
	}
	// delims can be overridden page by page.
	delimLeft, delimRight := page.DelimLeft, page.DelimRight
	if delimLeft == "" || delimRight == "" {
		delimLeft, delimRight = site.DelimLeft, site.DelimRight
	}
	tpl, err = tpl.Delims(delimLeft, delimRight).ParseFiles(files...)
	if err != nil {
		log.Printf("error: parse template, err: %v\n", err)
		return nil, err
	}
	site.templates[name] = tpl
	return tpl, nil
}

func (site *Site) handleError(rw http.ResponseWriter, r *http.Request, err error) {
	name := PageError
	if t, ok := site.Errors[ErrorFromErr(err).Code()]; ok {
		name = t
	}
	if err := site.handlePage(rw, r, name, site.GetPageData(name, r, err)); err != nil {
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
		data = site.GetPageData(name, r)
	}
	if err := t.Execute(w, data); err != nil {
		return err
	}
	return nil
}

// addEmbedPage add default pages, ready to use.
// Notes that default pages should use default delims [[]]
func (site *Site) addEmbedPage(name string, fs embed.FS, pattern string, p Page) {
	t, err := template.New(path.Base(pattern)).Delims(DefaultDelimeLeft, DefaultDelimeRight).Funcs(site.funcs).ParseFS(fs, pattern)
	if err != nil {
		log.Panic(err)
	}
	p.embed = true

	site.templates[name] = t
	site.Pages[name] = p
}

func (site *Site) addDefaultSiteMap() {
	site.addEmbedPage(PageSitemapXML, defaultTemplates, "templates/sitemap.xml", Page{
		Path: "/sitemap.xml",
		DataHandler: func(rw http.ResponseWriter, r *http.Request) interface{} {
			return SiteMap{
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
	site.addEmbedPage(PageRobotsTxt, defaultTemplates, "templates/robots.txt", Page{
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

// jsonFileDataHandler return DataHandler that read data from the given json file.
// Data can be accessed via .Data in templates.
// Panics if failed to read the file.
func (site *Site) jsonFileDataHandler(f string) DataHandler {
	loadData := func() (map[string]interface{}, error) {
		data := make(map[string]interface{})
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, Error(http.StatusInternalServerError, "read data from file, err: %v", err)
		}
		if err := json.Unmarshal(b, &data); err != nil {
			return nil, Error(http.StatusInternalServerError, "invalid data, err: %v", err)
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
	if site.Static != "" {
		if _, err := os.Stat(site.Static); err != nil {
			return fmt.Errorf("static path: %s, err: %w", site.Static, err)
		}
	}
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
