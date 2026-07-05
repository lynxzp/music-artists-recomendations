package server

import (
	"regexp"
	"unicode/utf8"
)

// safeInputPattern is an allowlist of printable characters: letters, numbers,
// space separators, punctuation, and symbols. It rejects only control/format
// characters (\p{C}) and non-space whitespace (tabs, newlines). This is
// permissive by design — real artist names use ':', '$', '+', '#', '|', dashes,
// etc. The value is only JSON-encoded for the proxy and structured-logged
// (never concatenated into SQL/HTML/shell), so there is no injection surface;
// the length cap in the callers bounds it.
var safeInputPattern = regexp.MustCompile(`^[\p{L}\p{N}\p{Zs}\p{P}\p{S}]+$`)

func isValidArtistName(s string) bool {
	return s == "" || (utf8.RuneCountInString(s) <= 256 && safeInputPattern.MatchString(s))
}

func isValidUsername(s string) bool {
	return s == "" || (utf8.RuneCountInString(s) <= 64 && safeInputPattern.MatchString(s))
}

func isValidPeriod(s string) bool {
	valid := map[string]bool{"": true, "overall": true, "7day": true, "1month": true, "3month": true, "6month": true, "12month": true}
	return valid[s]
}
