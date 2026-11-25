package main

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/runnelhq/sdk-go/runneltest"
)

func TestDescribe(t *testing.T) {
	c := runneltest.Start(t, newPlugin())
	if c.Initialized.Name != "iban" || c.Initialized.Version != version {
		t.Fatalf("initialize = %+v", c.Initialized)
	}
	d, err := c.Describe()
	if err != nil {
		t.Fatal(err)
	}
	var types []string
	for _, a := range d.Actions {
		types = append(types, a.Type)
	}
	if got := strings.Join(types, " "); got != "iban.validate iban.validate-bic iban.create" {
		t.Errorf("types = %s", got)
	}
}

// No type has a category, so that Runnel shows them in the group "Plugin: iban".
func TestNoCategories(t *testing.T) {
	c := runneltest.Start(t, newPlugin())
	raw, err := c.Call("describe", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `"category"`) {
		t.Errorf("describe names a category: %s", raw)
	}
}

func TestValidate(t *testing.T) {
	c := runneltest.Start(t, newPlugin())
	var out ibanOutput
	if err := c.ExecuteInto("iban.validate", map[string]any{"iban": "de89 3704 0044 0532 0130 00"}, &out); err != nil {
		t.Fatal(err)
	}
	want := ibanOutput{Valid: true, IBAN: "DE89370400440532013000", Formatted: "DE89 3704 0044 0532 0130 00", Country: "DE", CheckDigits: "89", BBAN: "370400440532013000", BankCode: "37040044"}
	if out != want {
		t.Errorf("output = %+v", out)
	}
	out = ibanOutput{}
	if err := c.ExecuteInto("iban.validate", map[string]any{"iban": " DE89370400440532013001 "}, &out); err != nil {
		t.Fatal(err)
	}
	if out.Valid || out.IBAN != "DE89370400440532013001" || !strings.Contains(out.Error, "check digits") {
		t.Errorf("output = %+v", out)
	}
	_, err := c.Execute("iban.validate", map[string]any{"iban": "DE00", "failOnInvalid": "true"})
	if err == nil || !strings.HasPrefix(err.Error(), "invalid IBAN: ") {
		t.Errorf("err = %v", err)
	}
}

func TestValidateBIC(t *testing.T) {
	c := runneltest.Start(t, newPlugin())
	var out bicOutput
	if err := c.ExecuteInto("iban.validate-bic", map[string]any{"bic": "COBADEFFXXX"}, &out); err != nil {
		t.Fatal(err)
	}
	if !out.Valid || out.Bank != "COBA" || out.Country != "DE" || out.Location != "FF" || out.Branch != "XXX" {
		t.Errorf("output = %+v", out)
	}
	if _, err := c.Execute("iban.validate-bic", map[string]any{"bic": "COBA", "failOnInvalid": true}); err == nil || !strings.Contains(err.Error(), "8 or 11") {
		t.Errorf("err = %v", err)
	}
}

func TestCreate(t *testing.T) {
	c := runneltest.Start(t, newPlugin())
	tests := []struct {
		config map[string]any
		want   string
		err    string
	}{
		{map[string]any{"country": "DE", "bankCode": "37040044", "accountNumber": "532013000"}, "DE89370400440532013000", ""},
		{map[string]any{"country": "de", "bban": "3704 0044 0532 0130 00"}, "DE89370400440532013000", ""},
		{map[string]any{"country": "AT", "bankCode": "19043", "accountNumber": "234573201"}, "AT611904300234573201", ""},
		{map[string]any{"country": "GB", "bban": "NWBK60161331926819"}, "GB29NWBK60161331926819", ""},
		{map[string]any{"country": "FR", "bankCode": "20041", "accountNumber": "1"}, "", "supported for DE, AT, CH and LI"},
		{map[string]any{"country": "DE", "bankCode": "3704004", "accountNumber": "1"}, "", "has 8 digits, not 7"},
		{map[string]any{"country": "DE", "bankCode": "37040044", "accountNumber": "12345678901"}, "", "at most 10"},
		{map[string]any{"country": "DE", "bankCode": "37040044", "accountNumber": "1", "bban": "1"}, "", "not both"},
		{map[string]any{"country": "DE"}, "", "give bban"},
		{map[string]any{"country": "DE", "bban": "37040044053201300X"}, "", "a digit expected"},
	}
	for _, tt := range tests {
		var out ibanOutput
		err := c.ExecuteInto("iban.create", tt.config, &out)
		if tt.err != "" {
			if err == nil || !strings.Contains(err.Error(), tt.err) {
				t.Errorf("%v: err = %v, want %q", tt.config, err, tt.err)
			}
			continue
		}
		if err != nil || out.IBAN != tt.want || !out.Valid {
			t.Errorf("%v: %+v, %v", tt.config, out, err)
		}
	}
}

func TestVersionMatchesPluginYAML(t *testing.T) {
	data, err := os.ReadFile("plugin.yaml")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^version:\s*(\S+)`).FindSubmatch(data)
	if m == nil || string(m[1]) != version {
		t.Errorf("plugin.yaml has version %q, the program %q", m, version)
	}
}

// TestWithRunnel runs the plugin in workflows of Runnel (needs RUNNEL_BIN).
func TestWithRunnel(t *testing.T) {
	p := runneltest.NewProject(t, ".", "plugin.yaml")
	p.WriteFile("workflows/check.workflow.yaml", `kind: Workflow
apiVersion: runnel/v1
name: check
parameters:
  - name: IBAN
outputs:
  - name: COUNTRY
    value: "{{ results.check.output.country }}"
  - name: FORMATTED
    value: "{{ results.check.output.formatted }}"
actions:
  - id: start
    type: core.start
  - id: check
    type: iban.validate
    with:
      iban: ${IBAN}
      failOnInvalid: true
links:
  - { from: start, to: check }
`)
	if out, code := p.Validate(); code != 0 {
		t.Fatalf("validate: exit %d\n%s", code, out)
	}
	out, code := p.Run("workflows/check.workflow.yaml", "IBAN=GB29 NWBK 6016 1331 9268 19")
	if code != 0 || !strings.Contains(out, `output COUNTRY = "GB"`) || !strings.Contains(out, `output FORMATTED = "GB29 NWBK 6016 1331 9268 19"`) {
		t.Fatalf("run: exit %d\n%s", code, out)
	}
	out, code = p.Run("workflows/check.workflow.yaml", "IBAN=GB29 NWBK 6016 1331 9268 18")
	if code != 1 || !strings.Contains(out, "invalid IBAN: the check digits are wrong") {
		t.Fatalf("run with invalid IBAN: exit %d\n%s", code, out)
	}
	// Runnel checks required fields against the schema from describe before the run.
	p.WriteFile("workflows/missing.workflow.yaml", `kind: Workflow
apiVersion: runnel/v1
name: missing
actions:
  - id: start
    type: core.start
  - id: check
    type: iban.validate
    with: {}
links:
  - { from: start, to: check }
`)
	if out, code := p.Validate(); code == 0 || !strings.Contains(out, "iban") {
		t.Fatalf("validate without iban: exit %d\n%s", code, out)
	}
}
