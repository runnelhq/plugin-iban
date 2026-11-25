package main

import (
	"context"
	"fmt"
	"slices"
	"strings"

	runnel "github.com/runnelhq/sdk-go"

	"github.com/runnelhq/plugin-iban/internal/iban"
)

// validateFieldConfig is the configuration of the transform iban.validate.
type validateFieldConfig struct {
	Field           string `json:"field" doc:"Field with the IBAN" schema:"required,format=runnel-field,example=iban"`
	Format          string `json:"format" doc:"How a valid IBAN is written back: electronic (DE89370400440532013000), print (groups of four) or keep (as read)" schema:"enum=electronic|print|keep,default=electronic"`
	CountryField    string `json:"countryField" doc:"New field for the country code, e.g. iban_country; empty: none"`
	BankCodeField   string `json:"bankCodeField" doc:"New field for the bank code (e.g. the BLZ), e.g. iban_bank; empty: none"`
	BranchCodeField string `json:"branchCodeField" doc:"New field for the branch code, e.g. iban_branch; empty: none"`
	Invalid         string `json:"invalid" doc:"Transform that records with an invalid IBAN go to, unchanged; empty: the error link with _error and _errorField" schema:"format=runnel-target"`
	AllowEmpty      bool   `json:"allowEmpty" doc:"Pass records with an empty IBAN on unchanged instead of treating them as invalid"`
}

// addedFields are the new fields by path in the configuration, in their order.
func (c validateFieldConfig) addedFields() [][2]string {
	var out [][2]string
	for _, f := range [][2]string{{"countryField", c.CountryField}, {"bankCodeField", c.BankCodeField}, {"branchCodeField", c.BranchCodeField}} {
		if f[1] != "" {
			out = append(out, f)
		}
	}
	return out
}

func outputFields(in runnel.Fields, c validateFieldConfig) runnel.Fields {
	out := slices.Clone(in)
	for _, f := range c.addedFields() {
		out = append(out, runnel.Field{Name: f[1], Type: runnel.TypeString})
	}
	return out
}

// validateRecords checks the IBAN in a field of every record.
func validateRecords(ctx context.Context, s *runnel.Stream, c validateFieldConfig) error {
	added := len(c.addedFields())
	for {
		b, ok, err := s.Next(ctx)
		if err != nil || !ok {
			return err
		}
		at := b.Fields.Index(c.Field)
		if at < 0 {
			return fmt.Errorf("field %q is not in the records", c.Field)
		}
		var valid, invalid, errs [][]any
		for _, row := range b.Rows {
			text := valueText(row[at])
			if strings.TrimSpace(text) == "" && c.AllowEmpty {
				valid = append(valid, append(row, make([]any, added)...))
				continue
			}
			i, err := iban.Parse(text)
			if err != nil {
				if c.Invalid != "" {
					invalid = append(invalid, row)
				} else {
					errs = append(errs, runnel.ErrorRow(row, "invalid IBAN: "+err.Error(), c.Field))
				}
				continue
			}
			switch c.Format {
			case "", "electronic":
				row[at] = i.IBAN
			case "print":
				row[at] = i.Formatted()
			}
			if c.CountryField != "" {
				row = append(row, i.Country)
			}
			if c.BankCodeField != "" {
				row = append(row, i.BankCode)
			}
			if c.BranchCodeField != "" {
				row = append(row, nullIfEmpty(i.BranchCode))
			}
			valid = append(valid, row)
		}
		if err := s.Emit(ctx, runnel.Main, outputFields(b.Fields, c), valid); err != nil {
			return err
		}
		if err := s.Emit(ctx, c.Invalid, b.Fields, invalid); err != nil {
			return err
		}
		if err := s.Emit(ctx, runnel.Error, runnel.ErrorFields(b.Fields), errs); err != nil {
			return err
		}
	}
}

func valueText(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	default:
		return fmt.Sprint(x)
	}
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// initValidate checks the configuration against the input and names the new fields.
func initValidate(_ context.Context, in runnel.InitInput, c validateFieldConfig) runnel.InitResult {
	var res runnel.InitResult
	problem := func(path, msg string) { res.Problems = append(res.Problems, runnel.Problem{Path: path, Message: msg}) }
	if c.Field != "" && !in.Input.Open && in.Input.Fields.Index(c.Field) < 0 {
		problem("field", fmt.Sprintf("field %q is not in the input", c.Field))
	}
	seen := map[string]string{}
	for _, f := range c.addedFields() {
		path, name := f[0], f[1]
		if in.Input.Fields.Index(name) >= 0 {
			problem(path, fmt.Sprintf("field %q exists already in the input", name))
		}
		if other, dup := seen[name]; dup {
			problem(path, fmt.Sprintf("%q is also the name in %s", name, other))
		}
		seen[name] = path
	}
	if c.Invalid != "" && !slices.Contains(in.Targets, c.Invalid) {
		problem("invalid", fmt.Sprintf("no data link leads to %q", c.Invalid))
	}
	res.Output = &runnel.RecordSchema{Fields: outputFields(in.Input.Fields, c), Open: in.Input.Open}
	return res
}
