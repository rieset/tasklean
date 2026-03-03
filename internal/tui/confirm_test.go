package tui

import (
	"strings"
	"testing"
)

func TestConfirmWithInput_Yes(t *testing.T) {
	result, err := ConfirmWithInput("Remove?", strings.NewReader("y"))
	if err != nil {
		t.Fatalf("ConfirmWithInput: %v", err)
	}
	if !result {
		t.Error("expected true for 'y' input")
	}
}

func TestConfirmWithInput_No(t *testing.T) {
	result, err := ConfirmWithInput("Remove?", strings.NewReader("n"))
	if err != nil {
		t.Fatalf("ConfirmWithInput: %v", err)
	}
	if result {
		t.Error("expected false for 'n' input")
	}
}

func TestConfirmWithInput_Enter(t *testing.T) {
	result, err := ConfirmWithInput("Remove?", strings.NewReader("\r"))
	if err != nil {
		t.Fatalf("ConfirmWithInput: %v", err)
	}
	if !result {
		t.Error("expected true for 'enter' input")
	}
}
