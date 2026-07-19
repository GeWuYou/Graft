// Package main 提供跨边界契约 TypeScript 派生产物的显式生成入口。
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"graft/server/internal/contract/projection"
)

const (
	defaultOutputDir    = "../web/src/contracts/generated"
	outputDirectoryMode = 0o750
	outputFileMode      = 0o600
)

// main 从 server 的 canonical Go contract 索引生成或校验 web 派生产物。
func main() {
	outputDir := flag.String("output-dir", defaultOutputDir, "directory for generated TypeScript outputs")
	check := flag.Bool("check", false, "compare generated content without writing")
	flag.Parse()

	for _, target := range projection.Targets() {
		content, err := projection.RenderTypeScript(target.Entries)
		if err != nil {
			failf("render contract projection %s: %v", target.Path, err)
		}
		outputPath := filepath.Join(*outputDir, target.Path)
		if *check {
			// outputPath 仅由固定 generated 根目录和 Targets 返回的仓库内相对路径构成。
			checkedIn, err := os.ReadFile(outputPath) // #nosec G304
			if err != nil {
				failf("read generated contract projection %s: %v", outputPath, err)
			}
			if !bytes.Equal(content, checkedIn) {
				failf("generated contract projection is stale: run `cd server && go run ./internal/contract/projection/cmd/projectiongen`")
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(outputPath), outputDirectoryMode); err != nil {
			failf("create generated contract projection directory: %v", err)
		}
		if err := os.WriteFile(outputPath, content, outputFileMode); err != nil {
			failf("write generated contract projection: %v", err)
		}
	}
}

func failf(format string, args ...any) {
	err := fmt.Errorf(format, args...)
	if errors.Is(err, flag.ErrHelp) {
		return
	}
	_, _ = fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
