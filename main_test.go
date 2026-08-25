package main

import "testing"

func TestHumanBytesUnmarshalFlag(t *testing.T) {
	var hb HumanBytes
	if err := hb.UnmarshalFlag("10MB"); err != nil {
		t.Fatalf("UnmarshalFlag error: %v", err)
	}
	if hb != 10*1000*1000 {
		t.Errorf("HumanBytes = %d; want 10000000", hb)
	}
}

func TestHumanBytesUnmarshalFlagInvalid(t *testing.T) {
	var hb HumanBytes
	if err := hb.UnmarshalFlag("invalid"); err == nil {
		t.Error("UnmarshalFlag should return error for invalid input")
	}
}
