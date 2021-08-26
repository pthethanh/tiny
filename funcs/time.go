package funcs

import (
	"fmt"
	"time"
)

func TimeFuncMap() map[string]interface{} {
	return map[string]interface{}{
		"date":     FormatTime,
		"duration": FormatDuration,
	}
}

// FormatTime format the given date
//
// Date can be a `time.Time` or an `int, int32, int64`.
// In the later case, it is treated as seconds since UNIX
// epoch.
func FormatTime(fmt string, zone string, date interface{}) string {
	if zone == "" {
		zone = "Local"
	}
	return formatDate(fmt, date, zone)
}

func formatDate(fmt string, date interface{}, zone string) string {
	var t time.Time
	switch date := date.(type) {
	default:
		t = time.Now()
	case time.Time:
		t = date
	case *time.Time:
		t = *date
	case int64:
		t = time.Unix(date, 0)
	case int:
		t = time.Unix(int64(date), 0)
	case int32:
		t = time.Unix(int64(date), 0)
	}

	loc, err := time.LoadLocation(zone)
	if err != nil {
		loc, _ = time.LoadLocation("UTC")
	}

	return t.In(loc).Format(fmt)
}

func FormatDuration(v interface{}) string {
	d := time.Duration(0)
	switch val := v.(type) {
	case time.Duration:
		d = val
	case int64:
		d = time.Duration(val)
	}
	rs := ""
	for d > 0 {
		switch {
		case d.Hours() >= 2:
			h := int(d.Hours())
			rs += fmt.Sprintf("%d hours ", h)
			d -= time.Duration(h) * time.Hour
		case d.Hours() >= 1:
			h := int(d.Hours())
			rs += fmt.Sprintf("%d hour ", h)
			d -= time.Duration(h) * time.Hour
		case d.Minutes() >= 2:
			m := int(d.Minutes())
			rs += fmt.Sprintf("%d minutes ", m)
			d -= time.Duration(m) * time.Minute
		case d.Minutes() >= 1:
			m := int(d.Minutes())
			rs += fmt.Sprintf("%d minute ", m)
			d -= time.Duration(m) * time.Minute
		case d.Seconds() >= 2:
			s := int(d.Seconds())
			rs += fmt.Sprintf("%d seconds", s)
			d -= time.Duration(s) * time.Second
		case d.Seconds() >= 1:
			s := int(d.Seconds())
			rs += fmt.Sprintf("%d second", s)
			d -= time.Duration(s) * time.Second
		default:
			return rs
		}
	}
	return rs
}
