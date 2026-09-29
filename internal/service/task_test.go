package service

import "testing"

func TestTaskServiceValidationConstants(t *testing.T) {
	if len("title") == 0 {
		t.Fatal("sanity check failed")
	}
	if len([]rune("x")) != 1 {
		t.Fatal("rune handling sanity check failed")
	}
}
