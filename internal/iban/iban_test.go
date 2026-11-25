package iban

import (
	"strings"
	"testing"
)

// Example IBANs of the SWIFT registry and of banks.
var valid = []string{
	"DE89370400440532013000", "DE02120300000000202051", "GB29NWBK60161331926819", "FR1420041010050500013M02606",
	"NL91ABNA0417164300", "BE68539007547034", "CH9300762011623852957", "AT611904300234573201",
	"ES9121000418450200051332", "IT60X0542811101000000123456", "PL61109010140000071219812874", "NO9386011117947",
	"SE4550000000058398257466", "DK5000400440116243", "FI2112345600000785", "LU280019400644750000",
	"IE29AIBK93115212345678", "PT50000201231234567890154", "MT84MALT011000012345MTLCAST001S",
	"SA0380000000608010167519", "TR330006100519786457841326", "BR1800360305000010009795493C1",
	"MU17BOMM0101101030300200000MUR", "KW81CBKU0000000000001234560101", "LI21088100002324013AA",
	"HR1210010051863000160", "CZ6508000000192000145399", "SK3112000000198742637541", "HU42117730161111101800000000",
	"RO49AAAA1B31007593840000", "BG80BNBG96611020345678", "GR1601101250000000012300695", "CY17002001280000001200527600",
	"SI56263300012039086", "LT121000011101001000", "LV80BANK0000435195001", "EE382200221020145685",
	"IS140159260076545510730339", "AD1200012030200359100100", "MC5811222000010123456789030", "SM86U0322509800000000270100",
	"GI75NWBK000000007099453", "VA59001123000012345678", "AL47212110090000000235698741", "BA391290079401028494",
	"ME25505000012345678951", "MK07250120000058984", "RS35260005601001611379", "GE29NB0000000101904917",
	"AZ21NABZ00000000137010001944", "KZ86125KZT5004100100", "UA213223130000026007233566001", "BY13NBRB3600900000002Z00AB00",
	"MD24AG000225100013104168", "IL620108000000099999999", "JO94CBJO0010000000000131000302", "LB62099900000001001901229114",
	"QA58DOHB00001234567890ABCDEFG", "AE070331234567890123456", "BH67BMAG00001299123456", "PS92PALS000000000400123456702",
	"PK36SCBL0000001123456702", "TN5910006035183598478831", "EG380019000500000000263180002", "CR05015202001026284066",
	"DO28BAGR00000001212453611324", "GT82TRAJ01020000001210029690", "SV62CENR00000000000000700025", "VG96VPVG0000012345678901",
	"XK051212012345678906", "TL380080012345678910157", "LC55HEMM000100010012001200023015", "ST68000100010051845310112",
	"SC18SSCB11010000000000001497USD", "IQ98NBIQ850123456789012", "BI4210000100010000332045181", "DJ2100010000000154000100186",
	"LY83002048000020100120361", "SD2129010501234001", "RU0304452522540817810538091310419", "MR1300020001010000123456753",
	"FO6264600001631634", "GL8964710001000206", "SO211000001001000100141", "NI45BAPR00000013000003558124",
	"MN121234123456789123", "OM810180000001299123456", "HN88CABF00000000000250005469", "FK88SC123456789012", "YE15CBYE0001018861234567891234",
}

func TestParseValid(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range valid {
		i, err := Parse(s)
		if err != nil {
			t.Errorf("%s: %v", s, err)
			continue
		}
		seen[i.Country] = true
		if i.IBAN != s || i.Country != s[:2] || i.BBAN != s[4:] || len(i.BankCode) == 0 {
			t.Errorf("%s: %+v", s, i)
		}
		if again, err := Create(i.Country, i.BBAN); err != nil || again.IBAN != s {
			t.Errorf("Create(%s, %s) = %v, %v", i.Country, i.BBAN, again.IBAN, err)
		}
	}
	for _, cc := range Countries() {
		if !seen[cc] {
			t.Errorf("no example for %s", cc)
		}
	}
}

func TestParts(t *testing.T) {
	tests := []struct{ in, bank, branch, formatted string }{
		{"de89 3704 0044 0532 0130 00", "37040044", "", "DE89 3704 0044 0532 0130 00"},
		{"IBAN GB29-NWBK-6016-1331-9268-19", "NWBK", "601613", "GB29 NWBK 6016 1331 9268 19"},
		{"IT60X0542811101000000123456", "05428", "11101", "IT60 X054 2811 1010 0000 0123 456"},
		{"FR1420041010050500013M02606", "20041", "01005", "FR14 2004 1010 0505 0001 3M02 606"},
	}
	for _, tt := range tests {
		i, err := Parse(tt.in)
		if err != nil || i.BankCode != tt.bank || i.BranchCode != tt.branch || i.Formatted() != tt.formatted {
			t.Errorf("%s: %+v %q, %v", tt.in, i, i.Formatted(), err)
		}
	}
}

func TestParseInvalid(t *testing.T) {
	tests := map[string]string{
		"":                            "the IBAN is empty",
		"DE89370400440532013001":      "the check digits are wrong",
		"DE98370400440532013000":      "the check digits are wrong",
		"DE8937040044053201300":       "an IBAN of DE has 22 characters, not 21",
		"XX89370400440532013000":      "country XX has no IBAN",
		"US12345678901234567890":      "country US has no IBAN",
		"DE89 3704 0044 0532 0130 0Ä": `character 22 ('Ä') is not a letter or digit`,
		"NL91ABN10417164300":          `character 8 ('1') does not fit an IBAN of NL: a letter expected`,
		"8989370400440532013000":      "an IBAN starts with a country code",
		"DE":                          "an IBAN starts with a country code",
		"CH00000000000AA1CCC00":       "the check digits 00 are not possible",
		strings.Repeat("D", 40):       "the IBAN has 40 characters",
	}
	for in, want := range tests {
		if _, err := Parse(in); err == nil || !strings.HasPrefix(err.Error(), want) {
			t.Errorf("Parse(%q) = %v, want %q", in, err, want)
		}
	}
}

func TestCreate(t *testing.T) {
	i, err := Create("de", "370400440532013000")
	if err != nil || i.IBAN != "DE89370400440532013000" {
		t.Errorf("Create = %+v, %v", i, err)
	}
	if _, err := Create("DE", "37040044053201300"); err == nil {
		t.Error("short BBAN accepted")
	}
	if _, err := Create("US", "1"); err == nil {
		t.Error("US accepted")
	}
	if Length("DE") != 22 || Length("NO") != 15 || Length("RU") != 33 || Length("US") != 0 {
		t.Error("Length")
	}
}

func TestParseBIC(t *testing.T) {
	b, err := ParseBIC("deut de ff 500")
	if err != nil || b.BIC != "DEUTDEFF500" || b.Bank != "DEUT" || b.Country != "DE" || b.Location != "FF" || b.Branch != "500" || b.Test {
		t.Errorf("ParseBIC = %+v, %v", b, err)
	}
	b, err = ParseBIC("NWBKGB20")
	if err != nil || b.Branch != "XXX" || !b.Test {
		t.Errorf("ParseBIC = %+v, %v", b, err)
	}
	for in, want := range map[string]string{
		"":            "the BIC is empty",
		"DEUTDEF":     "a BIC has 8 or 11 characters, not 7",
		"DEUTQQFF":    "characters 5 and 6 (QQ) are not a country code",
		"DEUT12FF":    "characters 5 and 6 (12) are not a country code",
		"DEUTDEFF_00": `character 9 ('_') is not a letter or digit`,
	} {
		if _, err := ParseBIC(in); err == nil || err.Error() != want {
			t.Errorf("ParseBIC(%q) = %v, want %q", in, err, want)
		}
	}
}

func FuzzParse(f *testing.F) {
	for _, s := range valid[:10] {
		f.Add(s)
	}
	f.Add("iban de89 3704-0044")
	f.Fuzz(func(t *testing.T, s string) {
		i, err := Parse(s)
		if err != nil {
			return
		}
		if mod97(i.BBAN+i.IBAN[:4]) != 1 || len(i.IBAN) != Length(i.Country) {
			t.Fatalf("accepted invalid IBAN %q", s)
		}
		if again, err := Create(i.Country, i.BBAN); err != nil || again != i {
			t.Fatalf("Create(%s) = %+v, %v; want %+v", i.BBAN, again, err, i)
		}
		_, _ = ParseBIC(s)
	})
}

func BenchmarkParse(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if _, err := Parse("DE89 3704 0044 0532 0130 00"); err != nil {
			b.Fatal(err)
		}
	}
}
