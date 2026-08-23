package service

import (
	"testing"

	"gotravel/internal/apperr"
)

func TestValidateUsername(t *testing.T) {
	if err := validateUsername("ab"); err == nil {
		t.Fatal("too short")
	}
	if err := validateUsername("good_user-1"); err != nil {
		t.Fatal(err)
	}
	if err := validateUsername("bad name"); !apperr.Is(err, apperr.Validation) {
		t.Fatal("space should fail")
	}
}
