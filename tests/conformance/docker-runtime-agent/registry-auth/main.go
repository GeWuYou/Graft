package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// main 使用仓库已锁定的 bcrypt 为任务私有 Registry 生成认证文件，秘密只从挂载文件读取。
func main() {
	username := flag.String("username", "fixture", "fixture registry username")
	passwordPath := flag.String("password-file", "/fixture/password", "restricted synthetic password file")
	output := flag.String("output", "/fixture/htpasswd", "restricted htpasswd output")
	flag.Parse()
	if err := writeAuthenticationFile(*username, *passwordPath, *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func writeAuthenticationFile(username, passwordPath, output string) error {
	if username == "" || strings.ContainsAny(username, ":\r\n") {
		return errors.New("fixture username is invalid")
	}
	password, err := os.ReadFile(passwordPath)
	if err != nil {
		return errors.New("read synthetic fixture password failed")
	}
	password = []byte(strings.TrimSpace(string(password)))
	if len(password) == 0 || len(password) > 72 {
		return errors.New("fixture password must contain 1 to 72 bytes")
	}
	hash, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	if err != nil {
		return errors.New("generate fixture password hash failed")
	}
	if err := os.WriteFile(output, []byte(username+":"+string(hash)+"\n"), 0o600); err != nil {
		return errors.New("write fixture authentication file failed")
	}
	return nil
}
