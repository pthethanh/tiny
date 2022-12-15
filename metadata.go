package tiny

import "fmt"

type (
	MetaData map[string]interface{}
)

func (m MetaData) GetStr(k string) string {
	v, ok := m[k]
	if !ok {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func (m MetaData) Version() string {
	return m.GetStr("version")
}

func (m MetaData) Lang() string {
	return m.GetStr("lang")
}

func (m MetaData) SiteName() string {
	return m.GetStr("site_name")
}

func (m MetaData) Title() string {
	return m.GetStr("title")
}

func (m MetaData) Domain() string {
	return m.GetStr("domain")
}

func (m MetaData) BaseURL() string {
	return m.GetStr("base_url")
}

func (m MetaData) CanonicalURL() string {
	return m.GetStr("canonical_url")
}

func (m MetaData) KeyWords() []string {
	v, ok := m["key_words"]
	if !ok {
		return []string{}
	}
	if v, ok := v.([]interface{}); ok {
		rs := []string{}
		for _, vv := range v {
			rs = append(rs, fmt.Sprintf("%v", vv))
		}
		return rs
	}
	if v, ok := v.([]string); ok {
		return v
	}
	return []string{fmt.Sprintf("%v", m["key_words"])}
}

func (m MetaData) Author() string {
	return m.GetStr("author")
}

func (m MetaData) Type() string {
	return m.GetStr("type")
}

func (m MetaData) Image() string {
	return m.GetStr("image")
}

func (m MetaData) Description() string {
	return m.GetStr("description")
}

func (m MetaData) SetVersion(v string) {
	m["version"] = v
}

func (m MetaData) SetLang(v string) {
	m["lang"] = v
}

func (m MetaData) SetSiteName(v string) {
	m["site_name"] = v
}

func (m MetaData) SetTitle(v string) {
	m["title"] = v
}

func (m MetaData) SetDomain(v string) {
	m["domain"] = v
}

func (m MetaData) SetBaseURL(v string) {
	m["base_url"] = v
}

func (m MetaData) SetCanonicalURL(v string) {
	m["canonical_url"] = v
}

func (m MetaData) SetKeyWords(v ...string) {
	m["key_words"] = v
}

func (m MetaData) SetAuthor(v string) {
	m["author"] = v
}

func (m MetaData) SetType(v string) {
	m["type"] = v
}

func (m MetaData) SetImage(v string) {
	m["image"] = v
}

func (m MetaData) SetDescription(v string) {
	m["description"] = v
}
