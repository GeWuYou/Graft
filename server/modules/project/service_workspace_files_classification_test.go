package project

import (
	"testing"

	projectcontract "graft/server/modules/project/contract"
)

func TestClassifyWorkspaceFile(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		path         string
		trackedKinds map[string]string
		wantFileKind string
		wantHint     string
		wantEditable bool
	}{
		{name: "jsonc extension", path: "config/app.jsonc", wantFileKind: "config", wantHint: "json", wantEditable: true},
		{name: "terraform extension", path: "infra/main.tf", wantFileKind: "config", wantHint: "hcl", wantEditable: true},
		{name: "terraform vars extension", path: "infra/dev.tfvars", wantFileKind: "config", wantHint: "hcl", wantEditable: true},
		{name: "powershell script", path: "scripts/bootstrap.ps1", wantFileKind: "config", wantHint: "powershell", wantEditable: true},
		{name: "powershell manifest", path: "scripts/module.psd1", wantFileKind: "config", wantHint: "powershell", wantEditable: true},
		{name: "zsh script", path: "scripts/env.zsh", wantFileKind: "config", wantHint: "shell", wantEditable: true},
		{name: "editorconfig filename", path: ".editorconfig", wantFileKind: "config", wantHint: "ini", wantEditable: true},
		{name: "gitconfig filename", path: ".gitconfig", wantFileKind: "config", wantHint: "ini", wantEditable: true},
		{name: "gitignore filename", path: ".gitignore", wantFileKind: "text", wantHint: "plaintext", wantEditable: false},
		{name: "gitattributes filename", path: ".gitattributes", wantFileKind: "text", wantHint: "plaintext", wantEditable: false},
		{name: "dockerfile filename", path: "Dockerfile", wantFileKind: "config", wantHint: "dockerfile", wantEditable: true},
		{name: "caddyfile filename", path: "Caddyfile", wantFileKind: "text", wantHint: "plaintext", wantEditable: false},
		{name: "makefile filename", path: "Makefile", wantFileKind: "text", wantHint: "plaintext", wantEditable: false},
		{name: "xml extension", path: "config/app.xml", wantFileKind: "config", wantHint: "xml", wantEditable: true},
		{name: "sql extension", path: "migrations/seed.sql", wantFileKind: "config", wantHint: "sql", wantEditable: true},
		{name: "markdown extension", path: "docs/README.md", wantFileKind: "text", wantHint: "markdown", wantEditable: false},
		{name: "dot env suffix", path: "config/.env.local", wantFileKind: "env", wantHint: "dotenv", wantEditable: true},
		{name: "trailing env suffix", path: "config/production.env", wantFileKind: "env", wantHint: "dotenv", wantEditable: true},
		{
			name:         "tracked env kind overrides unknown suffix",
			path:         "tracked/secrets.runtime",
			trackedKinds: map[string]string{"tracked/secrets.runtime": projectcontract.FileKindEnv.String()},
			wantFileKind: "env",
			wantHint:     "dotenv",
			wantEditable: true,
		},
		{
			name:         "tracked compose kind overrides unknown suffix",
			path:         "tracked/compose.runtime",
			trackedKinds: map[string]string{"tracked/compose.runtime": projectcontract.FileKindCompose.String()},
			wantFileKind: "compose",
			wantHint:     "yaml",
			wantEditable: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fileKind, hint, editable := classifyWorkspaceFile(tc.path, tc.trackedKinds)
			if fileKind != tc.wantFileKind || hint != tc.wantHint || editable != tc.wantEditable {
				t.Fatalf(
					"classifyWorkspaceFile(%q) = (%q, %q, %t), want (%q, %q, %t)",
					tc.path,
					fileKind,
					hint,
					editable,
					tc.wantFileKind,
					tc.wantHint,
					tc.wantEditable,
				)
			}
		})
	}
}
