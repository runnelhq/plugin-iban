// Package iban checks and builds International Bank Account Numbers (ISO 13616) and checks
// Business Identifier Codes (BIC, ISO 9362). The IBAN registry of SWIFT gives length and
// structure of the national part (BBAN) for each country.
package iban

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// IBAN is a checked IBAN and its parts.
type IBAN struct {
	IBAN        string // electronic format: capital letters, no spaces
	Country     string // ISO 3166 code, e.g. DE
	CheckDigits string // the two check digits
	BBAN        string // national part
	BankCode    string // bank code from the BBAN (e.g. the BLZ in Germany)
	BranchCode  string // branch code, empty where the country has none
}

// Formatted returns the IBAN in print format: groups of four characters.
func (i IBAN) Formatted() string { return Group(i.IBAN) }

// Group writes a text in groups of four characters, separated by spaces.
func Group(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 && i%4 == 0 {
			b.WriteByte(' ')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// ErrEmpty is the error for an empty IBAN.
var ErrEmpty = errors.New("the IBAN is empty")

// MaxLength is the longest IBAN of any country; longer input is rejected before it is
// checked further.
const MaxLength = 34

// Normalize removes spaces and hyphens and a leading "IBAN" and turns letters into capital
// letters, e.g. "iban de89 3704-0044 0532 0130 00" → "DE89370400440532013000".
func Normalize(s string) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || r == '-' {
			return -1
		}
		return unicode.ToUpper(r)
	}, s)
	return strings.TrimPrefix(s, "IBAN")
}

// Parse checks an IBAN (after Normalize): country, length and structure after the IBAN
// registry and the check digits (modulo 97).
func Parse(s string) (IBAN, error) {
	s = Normalize(s)
	if s == "" {
		return IBAN{}, ErrEmpty
	}
	if len(s) > MaxLength {
		return IBAN{}, fmt.Errorf("the IBAN has %d characters; an IBAN has at most %d", len(s), MaxLength)
	}
	for i, r := range s {
		if !isUpper(r) && !isDigit(r) {
			return IBAN{}, fmt.Errorf("character %d (%q) is not a letter or digit", i+1, r)
		}
	}
	if len(s) < 5 || !isUpper(rune(s[0])) || !isUpper(rune(s[1])) || !isDigit(rune(s[2])) || !isDigit(rune(s[3])) {
		return IBAN{}, errors.New("an IBAN starts with a country code and two check digits, e.g. DE89")
	}
	cc := s[:2]
	c, ok := countries[cc]
	if !ok {
		return IBAN{}, fmt.Errorf("country %s has no IBAN", cc)
	}
	bban := s[4:]
	if err := c.check(cc, bban); err != nil {
		return IBAN{}, err
	}
	if cd := s[2:4]; cd == "00" || cd == "01" || cd == "99" {
		return IBAN{}, fmt.Errorf("the check digits %s are not possible (02 to 98)", cd)
	}
	if mod97(bban+s[:4]) != 1 {
		return IBAN{}, errors.New("the check digits are wrong (mistyped or swapped characters)")
	}
	return c.parts(s), nil
}

// Create builds the IBAN of a country from its national part (BBAN) and computes the check
// digits.
func Create(cc, bban string) (IBAN, error) {
	cc = strings.ToUpper(strings.TrimSpace(cc))
	bban = Normalize(bban)
	c, ok := countries[cc]
	if !ok {
		return IBAN{}, fmt.Errorf("country %q has no IBAN", cc)
	}
	if err := c.check(cc, bban); err != nil {
		return IBAN{}, err
	}
	check := 98 - mod97(bban+cc+"00")
	return c.parts(fmt.Sprintf("%s%02d%s", cc, check, bban)), nil
}

// Countries returns the codes of all countries with an IBAN.
func Countries() []string {
	out := make([]string, 0, len(countries))
	for cc := range countries {
		out = append(out, cc)
	}
	return out
}

// Length returns the length of the IBAN of a country, or 0 if it has none.
func Length(cc string) int {
	c, ok := countries[cc]
	if !ok {
		return 0
	}
	return 4 + c.length()
}

func (c country) parts(s string) IBAN {
	bban := s[4:]
	i := IBAN{IBAN: s, Country: s[:2], CheckDigits: s[2:4], BBAN: bban}
	i.BankCode = bban[c.bank[0] : c.bank[0]+c.bank[1]]
	if c.branch[1] > 0 {
		i.BranchCode = bban[c.branch[0] : c.branch[0]+c.branch[1]]
	}
	return i
}

func (c country) length() int {
	n := 0
	for _, p := range c.pieces() {
		n += p.n
	}
	return n
}

type piece struct {
	n    int
	kind byte // n, a or c
}

func (c country) pieces() []piece {
	var out []piece
	num := 0
	for i := range len(c.bban) {
		ch := c.bban[i]
		if ch >= '0' && ch <= '9' {
			num = num*10 + int(ch-'0')
			continue
		}
		out = append(out, piece{num, ch})
		num = 0
	}
	return out
}

// check compares a BBAN with the structure of the country.
func (c country) check(cc, bban string) error {
	if want := c.length(); len(bban) != want {
		return fmt.Errorf("an IBAN of %s has %d characters, not %d", cc, want+4, len(bban)+4)
	}
	pos := 0
	for _, p := range c.pieces() {
		for _, r := range bban[pos : pos+p.n] {
			pos++
			ok := false
			switch p.kind {
			case 'n':
				ok = isDigit(r)
			case 'a':
				ok = isUpper(r)
			case 'c':
				ok = isDigit(r) || isUpper(r)
			}
			if !ok {
				return fmt.Errorf("character %d (%q) does not fit an IBAN of %s: %s expected", pos+4, r, cc, kindName(p.kind))
			}
		}
	}
	return nil
}

func kindName(kind byte) string {
	switch kind {
	case 'n':
		return "a digit"
	case 'a':
		return "a letter"
	}
	return "a letter or digit"
}

// mod97 computes the remainder of the number that s stands for, letters counted as 10 to 35
// (ISO 7064, MOD 97-10).
func mod97(s string) int {
	rest := 0
	for _, r := range s {
		var v int
		if isDigit(r) {
			v = int(r - '0')
			rest = (rest*10 + v) % 97
			continue
		}
		v = int(r-'A') + 10
		rest = (rest*100 + v) % 97
	}
	return rest
}

func isDigit(r rune) bool { return r >= '0' && r <= '9' }
func isUpper(r rune) bool { return r >= 'A' && r <= 'Z' }

// PadLeft fills a text with zeros on the left to n characters.
func PadLeft(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return strings.Repeat("0", n-len(s)) + s
}
