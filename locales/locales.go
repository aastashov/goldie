package locales

import (
	"encoding/json"
	"fmt"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

func GetBundle(baseDir string) (*i18n.Bundle, error) {
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	_, err := bundle.LoadMessageFile(baseDir + "locales/active.en.json")
	if err != nil {
		return nil, fmt.Errorf("load active.en.json: %w", err)
	}

	_, err = bundle.LoadMessageFile(baseDir + "locales/active.ru.json")
	if err != nil {
		return nil, fmt.Errorf("load active.ru.json: %w", err)
	}

	return bundle, nil
}
