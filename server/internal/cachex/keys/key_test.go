// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package keys

import "testing"

func TestNewRejectsColonSeparatedSegments(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		namespace string
		keyName   string
		parts     []string
	}{
		{
			name:      "namespace",
			namespace: "system:config",
			keyName:   "effective",
		},
		{
			name:      "name",
			namespace: "system-config",
			keyName:   "effective:auth",
		},
		{
			name:      "part",
			namespace: "system-config",
			keyName:   "effective",
			parts:     []string{"auth:admin"},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if _, err := New(testCase.namespace, testCase.keyName, testCase.parts...); err == nil {
				t.Fatal("expected colon-containing cache key segment to be rejected")
			}
		})
	}
}
