package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
)

type Language string

const (
	English Language = "en"
	Spanish Language = "es"
)

const (
	KeyInvalidCredentials     = "invalid_credentials"
	KeyInvalidToken           = "invalid_token"
	KeyForbidden              = "forbidden"
	KeyNotFound               = "not_found"
	KeyInvalidResetCode       = "invalid_reset_code"
	KeyResetExpired           = "reset_expired"
	KeyTooManyAttempts        = "too_many_attempts"
	KeyInvalidInput           = "invalid_input"
	KeyInternalServerError    = "internal_server_error"
	KeySignedOut              = "signed_out"
	KeyPasswordResetRequested = "password_reset_requested"
	KeyPasswordResetComplete  = "password_reset_complete"
)

type catalog struct {
	Messages    map[string]string `json:"messages"`
	Permissions map[string]string `json:"permissions"`
}

//go:embed locales/*.json
var localeFS embed.FS

var catalogs = mustLoadCatalogs()

func Normalize(value string) Language {
	for _, candidate := range strings.Split(value, ",") {
		current := strings.TrimSpace(candidate)
		if current == "" {
			continue
		}
		if index := strings.IndexByte(current, ';'); index >= 0 {
			current = current[:index]
		}
		current = strings.TrimSpace(strings.ToLower(current))
		if current == "" {
			continue
		}
		if index := strings.IndexAny(current, "-_"); index >= 0 {
			current = current[:index]
		}
		switch Language(current) {
		case Spanish:
			return Spanish
		case English:
			return English
		}
	}
	return English
}

func Message(language Language, key string) string {
	if message, ok := catalogs[language].Messages[key]; ok {
		return message
	}
	return catalogs[English].Messages[key]
}

func PermissionDescription(language Language, key, fallback string) string {
	if message, ok := catalogs[language].Permissions[key]; ok {
		return message
	}
	if fallback != "" {
		return fallback
	}
	return key
}

func mustLoadCatalogs() map[Language]catalog {
	return map[Language]catalog{
		English: mustLoadCatalog(English),
		Spanish: mustLoadCatalog(Spanish),
	}
}

func mustLoadCatalog(language Language) catalog {
	path := fmt.Sprintf("locales/%s.json", language)
	payload, err := localeFS.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("read i18n catalog %s: %v", path, err))
	}
	var current catalog
	if err := json.Unmarshal(payload, &current); err != nil {
		panic(fmt.Sprintf("decode i18n catalog %s: %v", path, err))
	}
	if current.Messages == nil {
		current.Messages = map[string]string{}
	}
	if current.Permissions == nil {
		current.Permissions = map[string]string{}
	}
	return current
}
