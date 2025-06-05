package main

import "testing"

func TestConvertToUnderscore(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"camel", "CamelCaseID", "camel_case_id", false},
		{"simple", "Simple", "simple", false},
		{"acronym", "HTTPServer", "http_server", false},
		{"invalid start", "1abc", "", true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := ConvertToUnderscore(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error for input %q", tc.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("ConvertToUnderscore(%q) returned error: %v", tc.in, err)
			}
			if got != tc.want {
				t.Errorf("ConvertToUnderscore(%q) = %q; want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestTrimInnerSpacesToOne(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"leading and trailing", "  Hello   world  ", "Hello world"},
		{"tabs and spaces", "a\t\tb   c", "a b c"},
		{"only spaces", "   ", ""},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := TrimInnerSpacesToOne(tc.in); got != tc.want {
				t.Errorf("TrimInnerSpacesToOne(%q) = %q; want %q", tc.in, got, tc.want)
			}
		})
	}
}
