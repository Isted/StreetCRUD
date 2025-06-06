package main

import "testing"

func TestConvertToUnderscore(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"CamelCaseID", "camel_case_id"},
		{"Simple", "simple"},
		{"HTTPServer", "http_server"},
		{"Camel_Case", "camel_case"},
	}
	for _, tt := range tests {
		got, err := ConvertToUnderscore(tt.in)
		if err != nil {
			t.Fatalf("ConvertToUnderscore(%q) returned error: %v", tt.in, err)
		}
		if got != tt.want {
			t.Errorf("ConvertToUnderscore(%q) = %q; want %q", tt.in, got, tt.want)
		}
	}
}

func TestTrimInnerSpacesToOne(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"  Hello   world  ", "Hello world"},
		{"a\t\tb   c", "a b c"},
		{"   ", ""},
	}
	for _, tt := range tests {
		if got := TrimInnerSpacesToOne(tt.in); got != tt.want {
			t.Errorf("TrimInnerSpacesToOne(%q) = %q; want %q", tt.in, got, tt.want)
		}
	}
}
