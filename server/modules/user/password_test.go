// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package user

import (
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestPasswordHasherHashesAndCompares(t *testing.T) {
	hasher := newPasswordHasher()

	hash, err := hasher.Hash("P@ssw0rd!")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if hash == "P@ssw0rd!" {
		t.Fatal("expected bcrypt hash instead of plaintext password")
	}

	if err := hasher.Compare(hash, "P@ssw0rd!"); err != nil {
		t.Fatalf("compare password: %v", err)
	}
}

func TestPasswordHasherRejectsMismatch(t *testing.T) {
	hasher := newPasswordHasher()

	hash, err := hasher.Hash("P@ssw0rd!")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	err = hasher.Compare(hash, "wrong-password")
	if !errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		t.Fatalf("expected bcrypt mismatch error, got %v", err)
	}
}

func TestPasswordHasherRejectsEmptyPassword(t *testing.T) {
	hasher := newPasswordHasher()

	if _, err := hasher.Hash(""); !errors.Is(err, errPasswordRequired) {
		t.Fatalf("expected empty password error on hash, got %v", err)
	}
	if err := hasher.Compare("hash", ""); !errors.Is(err, errPasswordRequired) {
		t.Fatalf("expected empty password error on compare, got %v", err)
	}
}
