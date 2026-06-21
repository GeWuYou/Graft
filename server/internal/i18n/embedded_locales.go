package i18n

import (
	"embed"
	"io/fs"
)

//go:embed locales/*.yaml locales/modules/*
var embeddedLocaleFiles embed.FS

var embeddedLocaleFS fs.FS = embeddedLocaleFiles
