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
	}{
		{name: "jsonc extension", path: "config/app.jsonc", wantFileKind: "config", wantHint: "json"},
		{name: "terraform extension", path: "infra/main.tf", wantFileKind: "config", wantHint: "hcl"},
		{name: "terraform vars extension", path: "infra/dev.tfvars", wantFileKind: "config", wantHint: "hcl"},
		{name: "powershell script", path: "scripts/bootstrap.ps1", wantFileKind: "config", wantHint: "powershell"},
		{name: "powershell manifest", path: "scripts/module.psd1", wantFileKind: "config", wantHint: "powershell"},
		{name: "zsh script", path: "scripts/env.zsh", wantFileKind: "config", wantHint: "shell"},
		{name: "editorconfig filename", path: ".editorconfig", wantFileKind: "config", wantHint: "ini"},
		{name: "gitconfig filename", path: ".gitconfig", wantFileKind: "config", wantHint: "ini"},
		{name: "gitignore filename", path: ".gitignore", wantFileKind: "text", wantHint: "plaintext"},
		{name: "gitattributes filename", path: ".gitattributes", wantFileKind: "text", wantHint: "plaintext"},
		{name: "dockerfile filename", path: "Dockerfile", wantFileKind: "config", wantHint: "dockerfile"},
		{name: "caddyfile filename", path: "Caddyfile", wantFileKind: "text", wantHint: "plaintext"},
		{name: "makefile filename", path: "Makefile", wantFileKind: "text", wantHint: "plaintext"},
		{name: "xml extension", path: "config/app.xml", wantFileKind: "config", wantHint: "xml"},
		{name: "sql extension", path: "migrations/seed.sql", wantFileKind: "config", wantHint: "sql"},
		{name: "markdown extension", path: "docs/README.md", wantFileKind: "text", wantHint: "markdown"},
		{name: "dot env suffix", path: "config/.env.local", wantFileKind: "env", wantHint: "dotenv"},
		{name: "trailing env suffix", path: "config/production.env", wantFileKind: "env", wantHint: "dotenv"},
		{
			name:         "tracked env kind overrides unknown suffix",
			path:         "tracked/secrets.runtime",
			trackedKinds: map[string]string{"tracked/secrets.runtime": projectcontract.FileKindEnv.String()},
			wantFileKind: "env",
			wantHint:     "dotenv",
		},
		{
			name:         "tracked compose kind overrides unknown suffix",
			path:         "tracked/compose.runtime",
			trackedKinds: map[string]string{"tracked/compose.runtime": projectcontract.FileKindCompose.String()},
			wantFileKind: "compose",
			wantHint:     "yaml",
		},
		{
			name:         "unknown suffix defaults to text",
			path:         "tracked/service.runtime",
			wantFileKind: "text",
			wantHint:     "plaintext",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fileKind, hint := classifyWorkspaceFile(tc.path, tc.trackedKinds)
			if fileKind != tc.wantFileKind || hint != tc.wantHint {
				t.Fatalf(
					"classifyWorkspaceFile(%q) = (%q, %q), want (%q, %q)",
					tc.path,
					fileKind,
					hint,
					tc.wantFileKind,
					tc.wantHint,
				)
			}
		})
	}
}
