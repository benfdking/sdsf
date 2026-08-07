package provider

func defaultIfEmpty(s string, defaultVal string) string {
	if s == "" {
		return defaultVal
	}
	return s
}
