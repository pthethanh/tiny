package tiny

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

// AuthInfo provides custom auth function for checking authentication status
// and retrieving authentication info.
func AuthInfo(f AuthInfoFunc) Option {
	return func(site *Site) {
		site.authInfo = f
	}
}
