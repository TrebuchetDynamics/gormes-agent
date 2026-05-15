package cli

import "strings"

// StripANSI removes terminal escape sequences from user-entered text.
func StripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		r, size := utf8DecodeRune(s[i:])
		if r == '\x1b' {
			i++
			if i < len(s) && s[i] == '[' {
				i++
				for i < len(s) {
					c := s[i]
					i++
					if (c >= 0x40 && c <= 0x7e) || c == '\x1b' {
						break
					}
				}
			}
			continue
		}
		b.WriteRune(r)
		i += size
	}
	return b.String()
}

func utf8DecodeRune(s string) (rune, int) {
	if len(s) == 0 {
		return -1, 0
	}
	if s[0] < 0x80 {
		return rune(s[0]), 1
	}
	r, size := rune(s[0]), 1
	for size < len(s) && s[size] >= 0x80 && s[size] < 0xc0 {
		r = (r << 6) | rune(s[size]&0x3f)
		size++
	}
	return r, size
}
