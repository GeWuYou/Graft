package auth

import (
	"unicode"
	"unicode/utf8"
)

const (
	defaultAdminUsername  = "graft"
	defaultAdminDisplay   = "Graft Admin"
	defaultAdminPassword  = "graft-admin"
	defaultAdminRoleName  = "admin"
	minimumPasswordLength = 12
)

type passwordPolicy struct{}

func newPasswordPolicy() passwordPolicy {
	return passwordPolicy{}
}

// ValidateNewPassword 校验密码长度、字母和数字组成，并拒绝使用默认管理员密码。
func (passwordPolicy) ValidateNewPassword(newPassword string) error {
	if newPassword == defaultAdminPassword {
		return errPasswordReuseForbidden
	}
	if utf8.RuneCountInString(newPassword) < minimumPasswordLength {
		return errPasswordPolicyViolation
	}

	var hasLetter bool
	var hasDigit bool
	for _, r := range newPassword {
		if unicode.IsLetter(r) {
			hasLetter = true
		}
		if unicode.IsDigit(r) {
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return errPasswordPolicyViolation
	}

	return nil
}
