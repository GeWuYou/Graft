// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package i18n

import (
	"embed"
	"io/fs"
)

//go:embed locales/*.yaml locales/modules/*
var embeddedLocaleFiles embed.FS

var embeddedLocaleFS fs.FS = embeddedLocaleFiles
