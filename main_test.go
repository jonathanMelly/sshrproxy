package main

import "testing"

func TestFindUpstreamByUsername(t *testing.T) {

	tests := []struct {
		name     string
		username string
		host     string
		wantErr  bool
	}{
		{"valid", "bob_test", "test", false},
		{"no underscore", "bobtpiorganiser", "", true},
		{"bad host", "bob_unknown", "", true},
		{"localhost", "bob_localhost", "", true},
	}
	separator = "_"
	suffix = ".blue.lan"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FindUpstreamByUsername(tt.username)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindUpstreamByUsername() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.host {
				t.Errorf("FindUpstreamByUsername() got = %v, want %v", got, tt.host)
			}
		})
	}
}
