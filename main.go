// Command iban is the Runnel plugin "iban": it checks IBANs and BICs and builds IBANs from
// the national account number; iban.validate is also a transform for pipelines. Runnel starts it; see README.md.
package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	runnel "github.com/runnelhq/sdk-go"

	"github.com/runnelhq/plugin-iban/internal/iban"
)

// version must match version in plugin.yaml (checked by a test).
const version = "0.3.0"

type validateConfig struct {
	IBAN          string `json:"iban" doc:"The IBAN to check; spaces, hyphens and small letters are allowed, e.g. DE89 3704 0044 0532 0130 00" schema:"required,example=DE89370400440532013000"`
	FailOnInvalid bool   `json:"failOnInvalid" doc:"Fail the action if the IBAN is invalid; false: the result is in valid and error"`
}

type ibanOutput struct {
	Valid       bool   `json:"valid"`
	IBAN        string `json:"iban"`      // electronic format; the input as written if invalid
	Formatted   string `json:"formatted"` // groups of four
	Country     string `json:"country"`
	CheckDigits string `json:"checkDigits"`
	BBAN        string `json:"bban"`
	BankCode    string `json:"bankCode"`
	BranchCode  string `json:"branchCode"`
	Error       string `json:"error"`
}

type bicConfig struct {
	BIC           string `json:"bic" doc:"The BIC (SWIFT code) to check, 8 or 11 characters, e.g. DEUTDEFF500" schema:"required,example=DEUTDEFF"`
	FailOnInvalid bool   `json:"failOnInvalid" doc:"Fail the action if the BIC is invalid; false: the result is in valid and error"`
}

type bicOutput struct {
	Valid    bool   `json:"valid"`
	BIC      string `json:"bic"`
	Bank     string `json:"bank"`
	Country  string `json:"country"`
	Location string `json:"location"`
	Branch   string `json:"branch"`
	Test     bool   `json:"test"`
	Error    string `json:"error"`
}

type createConfig struct {
	Country       string `json:"country" doc:"Country code (ISO 3166), e.g. DE" schema:"required,example=DE"`
	BBAN          string `json:"bban" doc:"The complete national account number (BBAN); alternatively bankCode and accountNumber for DE, AT, CH and LI" schema:"example=370400440532013000"`
	BankCode      string `json:"bankCode" doc:"Bank code, e.g. the BLZ in Germany (8 digits); only with accountNumber" schema:"example=37040044"`
	AccountNumber string `json:"accountNumber" doc:"Account number; filled with zeros on the left; only with bankCode" schema:"example=532013000"`
}

// accountLayouts tells for countries whose BBAN is bank code and account number how long
// both are; the account number is filled with zeros.
var accountLayouts = map[string][2]int{
	"DE": {8, 10},
	"AT": {5, 11},
	"CH": {5, 12},
	"LI": {5, 12},
}

func newPlugin() *runnel.Plugin {
	p := runnel.New("iban", version)
	runnel.AddAction(p, runnel.Action[validateConfig]{
		Type:        "iban.validate",
		Label:       "Validate IBAN",
		Description: "Checks an IBAN (country, length, structure and check digits) and returns its parts",
		Example:     "type: iban.validate\nwith:\n  iban: ${IBAN}\n  failOnInvalid: true\n",
		Reference: runnel.Reference{
			Outputs: ibanOutputs("Whether the IBAN is valid", "The IBAN in electronic format, e.g. DE89370400440532013000; as written if invalid"),
			Errors: "Fails only with failOnInvalid when the IBAN is invalid (unknown country, wrong length or characters, " +
				"wrong check digits); otherwise the result has valid false and the reason in error. " +
				"National check digits of the account number and whether bank or account exist are not checked.",
		},
		Run: func(_ context.Context, _ *runnel.Call, c validateConfig) (any, error) {
			i, err := iban.Parse(c.IBAN)
			if err != nil {
				if c.FailOnInvalid {
					return nil, fmt.Errorf("invalid IBAN: %w", err)
				}
				return ibanOutput{IBAN: strings.TrimSpace(c.IBAN), Error: err.Error()}, nil
			}
			return outputOf(i), nil
		},
	})
	runnel.AddTransform(p, runnel.Transform[validateFieldConfig]{
		Type:         "iban.validate",
		Label:        "Validate IBANs",
		Description:  "Checks the IBAN in a field of every record; invalid ones go to the error link or a target",
		Example:      "type: iban.validate\nwith:\n  field: iban\n  format: print\n  bankCodeField: blz\n",
		Reference:    validateFieldReference,
		RecordErrors: true,
		Parallel:     true,
		Run:          validateRecords,
		Init:         initValidate,
	})
	runnel.AddAction(p, runnel.Action[bicConfig]{
		Type:        "iban.validate-bic",
		Label:       "Validate BIC",
		Description: "Checks the form of a BIC (SWIFT code) and returns its parts; whether the bank exists is not checked",
		Example:     "type: iban.validate-bic\nwith:\n  bic: ${BIC}\n",
		Reference: runnel.Reference{
			Outputs: []runnel.OutputField{
				{Name: "output.valid", Type: "boolean", Description: "Whether the BIC has a valid form"},
				{Name: "output.bic", Type: "string", Description: "The BIC in capitals without spaces; as written if invalid"},
				{Name: "output.bank", Type: "string", Description: "Bank code, the first 4 characters"},
				{Name: "output.country", Type: "string", Description: "Country code (ISO 3166), characters 5 and 6"},
				{Name: "output.location", Type: "string", Description: "Location code, characters 7 and 8"},
				{Name: "output.branch", Type: "string", Description: "Branch code, XXX for the head office (also for a BIC of 8 characters)"},
				{Name: "output.test", Type: "boolean", Description: "Whether it is a test BIC (second character of the location is 0)"},
				{Name: "output.error", Type: "string", Description: "Why the BIC is invalid; empty if valid"},
			},
			Errors: "Fails only with failOnInvalid when the BIC is invalid (not 8 or 11 letters and digits, no country code); " +
				"otherwise the result has valid false and the reason in error. Whether the bank exists is not checked.",
		},
		Run: func(_ context.Context, _ *runnel.Call, c bicConfig) (any, error) {
			b, err := iban.ParseBIC(c.BIC)
			if err != nil {
				if c.FailOnInvalid {
					return nil, fmt.Errorf("invalid BIC: %w", err)
				}
				return bicOutput{BIC: strings.TrimSpace(c.BIC), Error: err.Error()}, nil
			}
			return bicOutput{Valid: true, BIC: b.BIC, Bank: b.Bank, Country: b.Country, Location: b.Location, Branch: b.Branch, Test: b.Test}, nil
		},
	})
	runnel.AddAction(p, runnel.Action[createConfig]{
		Type:        "iban.create",
		Label:       "Create IBAN",
		Description: "Builds an IBAN from country and national account number (BBAN) or bank code and account number",
		Example:     "type: iban.create\nwith:\n  country: DE\n  bankCode: \"37040044\"\n  accountNumber: \"532013000\"\n",
		Reference: runnel.Reference{
			Outputs: ibanOutputs("Always true", "The new IBAN in electronic format"),
			Errors: "Fails when the country has no IBAN, when not exactly one of bban or bankCode with accountNumber is given, " +
				"when bankCode and accountNumber are given for a country other than DE, AT, CH and LI, " +
				"or when the bank code, the account number or the BBAN do not fit the country. " +
				"The standard rule is used; some German banks have own IBAN rules of the Bundesbank.",
		},
		Run: func(_ context.Context, call *runnel.Call, c createConfig) (any, error) {
			bban, err := bbanOf(c)
			if err != nil {
				return nil, err
			}
			i, err := iban.Create(c.Country, bban)
			if err != nil {
				return nil, err
			}
			if i.Country == "DE" && c.BBAN == "" {
				call.Log.Debug("IBAN built by the standard rule; some German banks have own IBAN rules of the Bundesbank")
			}
			return outputOf(i), nil
		},
	})
	return p
}

func bbanOf(c createConfig) (string, error) {
	switch {
	case c.BBAN != "" && (c.BankCode != "" || c.AccountNumber != ""):
		return "", errors.New("give either bban or bankCode and accountNumber, not both")
	case c.BBAN != "":
		return c.BBAN, nil
	case c.BankCode == "" || c.AccountNumber == "":
		return "", errors.New("give bban, or bankCode and accountNumber")
	}
	cc := strings.ToUpper(strings.TrimSpace(c.Country))
	layout, ok := accountLayouts[cc]
	if !ok {
		return "", fmt.Errorf("bankCode and accountNumber are supported for DE, AT, CH and LI; give the bban for %s", cc)
	}
	bank := iban.Normalize(c.BankCode)
	account := iban.Normalize(c.AccountNumber)
	if len(bank) != layout[0] {
		return "", fmt.Errorf("the bank code of %s has %d digits, not %d", cc, layout[0], len(bank))
	}
	if len(account) > layout[1] {
		return "", fmt.Errorf("the account number of %s has at most %d characters, not %d", cc, layout[1], len(account))
	}
	return bank + iban.PadLeft(account, layout[1]), nil
}

// ibanOutputs describes ibanOutput, the result of iban.validate and iban.create.
func ibanOutputs(valid, ibanText string) []runnel.OutputField {
	return []runnel.OutputField{
		{Name: "output.valid", Type: "boolean", Description: valid},
		{Name: "output.iban", Type: "string", Description: ibanText},
		{Name: "output.formatted", Type: "string", Description: "The IBAN in groups of four, e.g. DE89 3704 0044 0532 0130 00"},
		{Name: "output.country", Type: "string", Description: "Country code, e.g. DE"},
		{Name: "output.checkDigits", Type: "string", Description: "The two check digits, e.g. 89"},
		{Name: "output.bban", Type: "string", Description: "The national account number (BBAN)"},
		{Name: "output.bankCode", Type: "string", Description: "Bank code, e.g. the BLZ in Germany"},
		{Name: "output.branchCode", Type: "string", Description: "Branch code; empty where the country has none"},
		{Name: "output.error", Type: "string", Description: "Why the IBAN is invalid; empty if valid"},
	}
}

func outputOf(i iban.IBAN) ibanOutput {
	return ibanOutput{
		Valid: true, IBAN: i.IBAN, Formatted: i.Formatted(), Country: i.Country, CheckDigits: i.CheckDigits,
		BBAN: i.BBAN, BankCode: i.BankCode, BranchCode: i.BranchCode,
	}
}

func main() { newPlugin().Main() }
