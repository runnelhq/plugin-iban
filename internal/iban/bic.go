package iban

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// BIC is a checked Business Identifier Code (ISO 9362, also SWIFT code).
type BIC struct {
	BIC      string // 8 or 11 characters, capital letters
	Bank     string // institution code, 4 characters (letters, since 2018 also digits)
	Country  string // ISO 3166 code
	Location string // location code, 2 characters
	Branch   string // branch code, 3 characters; "XXX" for the head office
	// Test reports a BIC for tests: the second character of the location is 0.
	Test bool
}

// ParseBIC checks a BIC: 8 or 11 letters and digits, a known country code at positions 5
// and 6. Spaces are removed and letters turned into capital letters. Whether the bank
// exists is not checked (that needs the directory of SWIFT).
func ParseBIC(s string) (BIC, error) {
	s = strings.ToUpper(strings.Join(strings.Fields(s), ""))
	if s == "" {
		return BIC{}, errors.New("the BIC is empty")
	}
	if len(s) != 8 && len(s) != 11 {
		return BIC{}, fmt.Errorf("a BIC has 8 or 11 characters, not %d", len(s))
	}
	for i, r := range s {
		if !isUpper(r) && !isDigit(r) {
			return BIC{}, fmt.Errorf("character %d (%q) is not a letter or digit", i+1, r)
		}
	}
	cc := s[4:6]
	if !isUpper(rune(cc[0])) || !isUpper(rune(cc[1])) || !slices.Contains(isoCountries, cc) {
		return BIC{}, fmt.Errorf("characters 5 and 6 (%s) are not a country code", cc)
	}
	b := BIC{BIC: s, Bank: s[:4], Country: cc, Location: s[6:8], Branch: "XXX", Test: s[7] == '0'}
	if len(s) == 11 {
		b.Branch = s[8:]
	}
	return b, nil
}

// isoCountries are the codes of ISO 3166-1 alpha-2 and XK (Kosovo), which SWIFT uses.
var isoCountries = strings.Fields(`
AD AE AF AG AI AL AM AO AQ AR AS AT AU AW AX AZ BA BB BD BE BF BG BH BI BJ BL BM BN BO BQ BR BS BT BV BW BY BZ
CA CC CD CF CG CH CI CK CL CM CN CO CR CU CV CW CX CY CZ DE DJ DK DM DO DZ EC EE EG EH ER ES ET FI FJ FK FM FO FR
GA GB GD GE GF GG GH GI GL GM GN GP GQ GR GS GT GU GW GY HK HM HN HR HT HU ID IE IL IM IN IO IQ IR IS IT JE JM JO
JP KE KG KH KI KM KN KP KR KW KY KZ LA LB LC LI LK LR LS LT LU LV LY MA MC MD ME MF MG MH MK ML MM MN MO MP MQ MR
MS MT MU MV MW MX MY MZ NA NC NE NF NG NI NL NO NP NR NU NZ OM PA PE PF PG PH PK PL PM PN PR PS PT PW PY QA RE RO
RS RU RW SA SB SC SD SE SG SH SI SJ SK SL SM SN SO SR SS ST SV SX SY SZ TC TD TF TG TH TJ TK TL TM TN TO TR TT TV
TW TZ UA UG UM US UY UZ VA VC VE VG VI VN VU WF WS YE YT ZA ZM ZW XK`)
