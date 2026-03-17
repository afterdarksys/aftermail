package i18n

import (
	"embed"
	"encoding/json"
	"log"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed locales/*.json
var localeFS embed.FS

var (
	bundle    *i18n.Bundle
	localizer *i18n.Localizer
	currLang  string
)

// Init sets up the i18n engine with the specified base language.
func Init(lang string) error {
	bundle = i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	// Load local translation files
	files, err := localeFS.ReadDir("locales")
	if err != nil {
		return err
	}

	for _, file := range files {
		if !file.IsDir() {
			_, err := bundle.LoadMessageFileFS(localeFS, "locales/"+file.Name())
			if err != nil {
				log.Printf("[i18n] Warning: Failed to load translation file %s: %v", file.Name(), err)
			}
		}
	}

	SetLanguage(lang)
	return nil
}

// SetLanguage updates the current active language engine.
func SetLanguage(lang string) {
	currLang = lang
	localizer = i18n.NewLocalizer(bundle, lang, "en")
}

// T translates a message ID into the currently active language.
func T(messageID string, defaultMessage string) string {
	if localizer == nil {
		return defaultMessage
	}

	translated, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID: messageID,
		DefaultMessage: &i18n.Message{
			ID:    messageID,
			Other: defaultMessage,
		},
	})
	
	if err != nil {
		return defaultMessage
	}
	return translated
}

// TParams translates a parameterized message.
func TParams(messageID string, defaultMessage string, templateData map[string]interface{}) string {
	if localizer == nil {
		return defaultMessage
	}

	translated, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID: messageID,
		DefaultMessage: &i18n.Message{
			ID:    messageID,
			Other: defaultMessage,
		},
		TemplateData: templateData,
	})

	if err != nil {
		return defaultMessage
	}
	return translated
}
