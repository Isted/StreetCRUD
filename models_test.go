package main

import "testing"

func TestCheckStructForDeletes(t *testing.T) {
	tests := []struct {
		name string
		cols []*column
		want bool
	}{
		{"none", []*column{}, true},
		{"deleted only", []*column{{deleted: true}}, false},
		{"deletedOn only", []*column{{deletedOn: true}}, false},
		{"both", []*column{{deleted: true}, {deletedOn: true}}, true},
		{"multiple deleted", []*column{{deleted: true}, {deleted: true}, {deletedOn: true}}, false},
		{"multiple deletedOn", []*column{{deleted: true}, {deletedOn: true}, {deletedOn: true}}, false},
		{"two each", []*column{{deleted: true}, {deletedOn: true}, {deleted: true}, {deletedOn: true}}, false},
	}
	for _, tt := range tests {
		s := &structToCreate{cols: tt.cols}
		if got := s.CheckStructForDeletes(); got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
}
