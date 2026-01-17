package server

import (
	"regexp"
	"unicode/utf8"
)

var safeInputPattern = regexp.MustCompile(`^[\p{L}\p{N}\p{Zs}\-_.'&]+$`)

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
