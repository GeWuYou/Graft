package user

import "unicode"

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

// ValidateNewPassword 校验一次新密码是否满足当前 MVP 固定策略。
func (passwordPolicy) ValidateNewPassword(newPassword string) error {
	if newPassword == defaultAdminPassword {
		return errPasswordReuseForbidden
	}
	if len(newPassword) < minimumPasswordLength {
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
