package server

import (
	"strings"
	"testing"
)

func TestIsValidArtistName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"empty string is valid", "", true},
		{"normal artist name", "Radiohead", true},
		{"artist with spaces", "The Beatles", true},
		{"unicode characters", "Бетховен", true},
		{"special chars allowed", "AC/DC", true},
		{"apostrophe allowed", "Guns N' Roses", true},
		{"ampersand allowed", "Simon & Garfunkel", true},
		{"parentheses allowed", "Crash Test (Band)", true},
		{"exclamation allowed", "Panic! At The Disco", true},
		{"double quote allowed", "\"Weird Al\" Yankovic", true},
		{"asterisk allowed", "THE*GA*GA*S", true},
		{"backtick allowed", "test" + "`" + "name", true},
		{"hyphen and underscore", "my-artist_name", true},
		{"period and comma", "Dr. Dre, Jr.", true},
		{"max length 256 runes", strings.Repeat("a", 256), true},
		{"over max length", strings.Repeat("a", 257), false},
		{"colon allowed", "Re:Zound", true},
		{"colon and periods allowed", "Sixx:A.M.", true},
		{"dollar sign allowed", "Ke$ha", true},
		{"plus allowed", "+/-", true},
		{"double quote valid", "artist\"name", true},
		// Punctuation/symbols are allowed: the value is only JSON-encoded for the
		// proxy and structured-logged, and the UI escapes names on render — no
		// SQL/HTML/shell injection surface.
		{"semicolon allowed", "artist;drop", true},
		{"angle brackets allowed", "artist<script>", true},
		{"control chars still rejected: newline", "artist\nname", false},
		{"control chars still rejected: tab", "artist\tname", false},
		{"numbers valid", "Blink182", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidArtistName(tt.input)
			if got != tt.want {
				t.Errorf("isValidArtistName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsValidUsername(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"empty string is valid", "", true},
		{"normal username", "john_doe", true},
		{"max length 64 runes", strings.Repeat("a", 64), true},
		{"over max length", strings.Repeat("a", 65), false},
		{"punctuation allowed", "user;name", true},
		{"angle brackets allowed", "user<name>", true},
		{"control chars still rejected: newline", "user\nname", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidUsername(tt.input)
			if got != tt.want {
				t.Errorf("isValidUsername(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsValidPeriod(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"empty string is valid", "", true},
		{"overall", "overall", true},
		{"7day", "7day", true},
		{"1month", "1month", true},
		{"3month", "3month", true},
		{"6month", "6month", true},
		{"12month", "12month", true},
		{"invalid period", "2month", false},
		{"random string", "foobar", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidPeriod(tt.input)
			if got != tt.want {
				t.Errorf("isValidPeriod(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
