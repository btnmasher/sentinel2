package eve

var supportedSystemLocales = []string{"ja", "ko", "zh"}

// SupportedSystemLocales returns the SDE locale keys stored on solar systems.
func SupportedSystemLocales() []string {
	return append([]string(nil), supportedSystemLocales...)
}

// SystemNameField returns the solar_systems field name for a supported locale.
func SystemNameField(locale string) (string, bool) {
	switch locale {
	case "ja":
		return "name_ja", true
	case "ko":
		return "name_ko", true
	case "zh":
		return "name_zh", true
	default:
		return "", false
	}
}
