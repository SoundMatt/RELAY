// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package relay

import (
	"errors"
	"fmt"
	"testing"
)

func TestSentinelsNotNil(t *testing.T) {
	sentinels := []struct {
		name string
		err  error
	}{
		{"ErrClosed", ErrClosed},
		{"ErrNotConnected", ErrNotConnected},
		{"ErrTimeout", ErrTimeout},
		{"ErrPayloadTooLarge", ErrPayloadTooLarge},
	}
	for _, s := range sentinels {
		if s.err == nil {
			t.Errorf("%s must not be nil", s.name)
		}
		if s.err.Error() == "" {
			t.Errorf("%s.Error() must not be empty", s.name)
		}
	}
}

func TestSentinelsDistinct(t *testing.T) {
	all := []error{ErrClosed, ErrNotConnected, ErrTimeout, ErrPayloadTooLarge}
	for i, a := range all {
		for j, b := range all {
			if i != j && errors.Is(a, b) {
				t.Errorf("sentinel %d and %d must be distinct", i, j)
			}
		}
	}
}

func TestErrorWrapping(t *testing.T) {
	cases := []error{ErrClosed, ErrNotConnected, ErrTimeout, ErrPayloadTooLarge}
	for _, sentinel := range cases {
		wrapped := fmt.Errorf("protocol layer: %w", sentinel)
		if !errors.Is(wrapped, sentinel) {
			t.Errorf("wrapped %v must satisfy errors.Is", sentinel)
		}
		doubleWrapped := fmt.Errorf("transport: %w", wrapped)
		if !errors.Is(doubleWrapped, sentinel) {
			t.Errorf("double-wrapped %v must satisfy errors.Is", sentinel)
		}
	}
}

func TestSentinelMessagePrefix(t *testing.T) {
	sentinels := []error{ErrClosed, ErrNotConnected, ErrTimeout, ErrPayloadTooLarge}
	for _, s := range sentinels {
		msg := s.Error()
		if len(msg) < 7 || msg[:7] != "relay: " {
			t.Errorf("%v message must start with \"relay: \", got %q", s, msg)
		}
	}
}
