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
	defaultOutputPath   = "../web/src/contracts/generated/platform.ts"
	outputDirectoryMode = 0o750
	outputFileMode      = 0o600
)

// main 从 server 的 canonical Go contract 索引生成或校验 web 派生产物。
func main() {
	outputPath := flag.String("output", defaultOutputPath, "path to the generated TypeScript output")
	check := flag.Bool("check", false, "compare generated content without writing")
	flag.Parse()

	content, err := projection.RenderTypeScript(projection.Registry())
	if err != nil {
		failf("render contract projection: %v", err)
	}
	if *check {
		checkedIn, err := os.ReadFile(*outputPath)
		if err != nil {
			failf("read generated contract projection %s: %v", *outputPath, err)
		}
		if !bytes.Equal(content, checkedIn) {
			failf("generated contract projection is stale: run `cd server && go run ./internal/contract/projection/cmd/projectiongen`")
		}
		return
	}

	if err := os.MkdirAll(filepath.Dir(*outputPath), outputDirectoryMode); err != nil {
		failf("create generated contract projection directory: %v", err)
	}
	if err := os.WriteFile(*outputPath, content, outputFileMode); err != nil {
		failf("write generated contract projection: %v", err)
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
