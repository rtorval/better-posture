package i18n

import (
	"os"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var matcher = language.NewMatcher([]language.Tag{
	language.English, // default
	language.Spanish, // es, es-ES
})

var (
	kernel32                  = windows.NewLazySystemDLL("kernel32.dll")
	procGetUserDefaultLocaleW = kernel32.NewProc("GetUserDefaultLocaleName")
)

func normalizeLocale(locale string) string {
	if locale == "" {
		return "en"
	}

	if idx := strings.Index(locale, "."); idx != -1 {
		locale = locale[:idx]
	}

	locale = strings.ReplaceAll(locale, "_", "-")

	parts := strings.Split(locale, "-")
	if len(parts) == 0 {
		return "en"
	}

	if len(parts) == 1 {
		return parts[0]
	}

	return parts[0] + "-" + strings.ToUpper(parts[1])
}

func DetectUserLanguage() string {
	if lang := os.Getenv("LANG"); lang != "" {
		return normalizeLocale(lang)
	}

	var buf [85]uint16
	ret, _, callErr := procGetUserDefaultLocaleW.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
	)

	if ret != 0 {
		n := int(ret)
		if n <= 0 {
		} else {
			if n > len(buf) {
				n = len(buf)
			}
			raw := windows.UTF16ToString(buf[:n])
			return normalizeLocale(raw)
		}
	} else {
		_ = callErr
	}

	return "en"
}

func Printer() *message.Printer {
	userLang := DetectUserLanguage()
	tag, _, _ := matcher.Match(language.Make(userLang))
	return message.NewPrinter(tag)
}
