package user

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

var errPasswordRequired = errors.New("password is required")

type passwordHasher struct {
	cost int
}

func newPasswordHasher() passwordHasher {
	return passwordHasher{cost: bcrypt.DefaultCost}
}

// Hash 使用 bcrypt 生成单向密码散列，避免插件保留明文口令。
func (h passwordHasher) Hash(password string) (string, error) {
	if password == "" {
		return "", errPasswordRequired
	}

	sum, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}

	return string(sum), nil
}

// Compare 使用 bcrypt 校验输入口令与已保存散列是否匹配。
func (h passwordHasher) Compare(hash string, password string) error {
	if password == "" {
		return errPasswordRequired
	}

	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
