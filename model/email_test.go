package model_test

import (
	"testing"

	"github.com/matryer/is"

	"canvas/model"
)

func TestEmail_IsValid(t *testing.T) {
	// This is table driven tests
	tests := []struct {
		address string
		valid   bool
	}{
		{"me@example.com", true},
		{"me@example", true},
		{"@example.com", false},
		{"me@", false},
		{"@", false},
		{"", false},
		{"me", false},
		{"me@example..example", false},
		{"me@example.", false},
	}
	t.Run("reports valid email addresses", func(t *testing.T) {
		for _, test := range tests {
			t.Run(test.address, func(t *testing.T) {
				is := is.New(t)
				e := model.Email(test.address)
				is.Equal(test.valid, e.IsValid())
			})
		}
	})
}
