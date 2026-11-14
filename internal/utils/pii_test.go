package utils

import "testing"

func TestMaskPhone(t *testing.T) {
	cases := map[string]string{
		"":              "",
		"9876543210":    "********10",
		"+919876543210": "+**********10",
		"12":            "**",
	}
	for in, want := range cases {
		if got := MaskPhone(in); got != want {
			t.Errorf("MaskPhone(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMaskName(t *testing.T) {
	cases := map[string]string{
		"":         "",
		"John":     "J***",
		"John Doe": "J*** D**",
		"A":        "A",
	}
	for in, want := range cases {
		if got := MaskName(in); got != want {
			t.Errorf("MaskName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMaskAddress(t *testing.T) {
	if got := MaskAddress("12 MG Road, Bengaluru, Karnataka 560001"); got != "[redacted address] PIN 560001" {
		t.Errorf("MaskAddress with PIN = %q", got)
	}
	if got := MaskAddress("No pincode here"); got != "[redacted address]" {
		t.Errorf("MaskAddress without PIN = %q", got)
	}
	if got := MaskAddress(""); got != "" {
		t.Errorf("MaskAddress empty = %q", got)
	}
}

func TestMaskPhonesInText(t *testing.T) {
	in := `{"name":"John","phone":"9876543210","note":"call +919812345678"}`
	got := MaskPhonesInText(in)
	want := `{"name":"John","phone":"********10","note":"call +**********78"}`
	if got != want {
		t.Errorf("MaskPhonesInText = %q, want %q", got, want)
	}
}
