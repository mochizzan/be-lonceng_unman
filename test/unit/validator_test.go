package unit_test

import (
	"testing"

	"be-lonceng_unman/internal/services"

	"github.com/stretchr/testify/assert"
)

func TestValidateNIS_Valid(t *testing.T) {
	validCases := []string{
		"2211700006",
		"1234567890",
		"0000000000",
		"9999999999",
	}
	for _, nis := range validCases {
		t.Run("valid_"+nis, func(t *testing.T) {
			err := services.ValidateNIS(nis)
			assert.NoError(t, err)
		})
	}
}

func TestValidateNIS_Invalid(t *testing.T) {
	invalidCases := []struct {
		name string
		nis  string
	}{
		{"empty", ""},
		{"too_short", "123"},
		{"9_digits", "221170000"},
		{"11_digits", "22117000061"},
		{"letters", "abcdefghij"},
		{"mixed_letters", "abc1234567"},
		{"symbols", "221170000!"},
		{"spaces", "221170 006"},
		{"dash", "221170-006"},
	}
	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			err := services.ValidateNIS(tc.nis)
			assert.Error(t, err)
		})
	}
}
