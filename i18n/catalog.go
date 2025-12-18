package i18n

import (
	"embed"
	"fmt"

	"github.com/BurntSushi/toml"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

//go:embed *.toml
var localeFS embed.FS

type translationMap map[string]string

func init() {
	loadLocale("en.toml", language.English)
	loadLocale("es.toml", language.Spanish)
}

func loadLocale(file string, lang language.Tag) {
	data, err := localeFS.ReadFile(file)
	if err != nil {
		fmt.Printf("Error reading locale file %s: %v\n", file, err)
		return
	}

	var m translationMap
	if _, err := toml.Decode(string(data), &m); err != nil {
		fmt.Printf("Error parsing locale file %s: %v\n", file, err)
		return
	}
	for k, v := range m {
		message.SetString(lang, k, v)
	}
}
