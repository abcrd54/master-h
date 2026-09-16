package main

import "testing"

func TestNormalizeAddressAddsDefaultADBPort(t *testing.T) {
	got, err := normalizeAddress("192.168.50.10")
	if err != nil || got != "192.168.50.10:5555" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestNormalizeAddressAcceptsSpecifiedPort(t *testing.T) {
	got, err := normalizeAddress("192.168.50.10:5037")
	if err != nil || got != "192.168.50.10:5037" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestNormalizeAddressRejectsInvalidInput(t *testing.T) {
	for _, input := range []string{"", "stb.local", "192.168.50.10:0", "192.168.50.10:99999"} {
		if _, err := normalizeAddress(input); err == nil {
			t.Errorf("invalid input %q accepted", input)
		}
	}
}
