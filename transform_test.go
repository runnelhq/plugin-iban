package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	runnel "github.com/runnelhq/sdk-go"
	"github.com/runnelhq/sdk-go/runneltest"
)

var accounts = runnel.Fields{{Name: "id", Type: runnel.TypeInt}, {Name: "iban", Type: runnel.TypeString}}

func TestValidateTransform(t *testing.T) {
	c := runneltest.Start(t, newPlugin())
	batch := runnel.Batch{Fields: accounts, Rows: [][]any{
		{1, "de89 3704 0044 0532 0130 00"},
		{2, "DE89370400440532013001"},
		{3, ""},
		{4, "GB29NWBK60161331926819"},
		{5, nil},
	}}
	res, err := c.RunTransform("iban.validate", map[string]any{"field": "iban", "format": "print", "countryField": "country", "bankCodeField": "bank", "branchCodeField": "branch"}, batch)
	if err != nil || res.Error != "" {
		t.Fatal(err, res.Error)
	}
	main, errs := res.Records(runnel.Main), res.Records(runnel.Error)
	if len(main) != 2 || main[0]["iban"] != "DE89 3704 0044 0532 0130 00" || main[0]["country"] != "DE" || main[0]["bank"] != "37040044" || main[0]["branch"] != nil ||
		main[1]["branch"] != "601613" {
		t.Errorf("main = %v", main)
	}
	if len(errs) != 3 || errs[0]["_errorField"] != "iban" || !strings.HasPrefix(errs[0]["_error"].(string), "invalid IBAN: the check digits are wrong") ||
		errs[1]["_error"] != "invalid IBAN: the IBAN is empty" || errs[0]["country"] != nil {
		t.Errorf("errors = %v", errs)
	}

	// Empty values pass, invalid ones go to a target unchanged.
	res, err = c.RunTransform("iban.validate", map[string]any{"field": "iban", "allowEmpty": true, "invalid": "check-by-hand"}, batch)
	if err != nil || res.Error != "" {
		t.Fatal(err, res.Error)
	}
	main, routed := res.Records(runnel.Main), res.Records("check-by-hand")
	if len(main) != 4 || main[0]["iban"] != "DE89370400440532013000" || len(routed) != 1 || routed[0]["iban"] != "DE89370400440532013001" || len(res.Records(runnel.Error)) != 0 {
		t.Errorf("main = %v, routed = %v", main, routed)
	}

	res, err = c.RunTransform("iban.validate", map[string]any{"field": "account"}, batch)
	if err != nil || res.Error != `field "account" is not in the records` {
		t.Errorf("missing field: %v %q", err, res.Error)
	}
}

func TestValidateInit(t *testing.T) {
	c := runneltest.Start(t, newPlugin())
	in := runnel.InitInput{Input: runnel.RecordSchema{Fields: accounts}, Targets: []string{"write"}}
	res, err := c.Init("iban.validate", map[string]any{"field": "iban", "bankCodeField": "blz"}, in)
	if err != nil || len(res.Problems) != 0 || res.Output == nil || len(res.Output.Fields) != 3 || res.Output.Fields[2] != (runnel.Field{Name: "blz", Type: runnel.TypeString}) {
		t.Errorf("init = %+v, %v", res, err)
	}
	res, err = c.Init("iban.validate", map[string]any{"field": "account", "countryField": "id", "bankCodeField": "x", "branchCodeField": "x", "invalid": "nowhere"}, in)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, p := range res.Problems {
		got = append(got, p.Path+": "+p.Message)
	}
	want := `field: field "account" is not in the input|countryField: field "id" exists already in the input|branchCodeField: "x" is also the name in bankCodeField|invalid: no data link leads to "nowhere"`
	if strings.Join(got, "|") != want {
		t.Errorf("problems\n got %s\nwant %s", strings.Join(got, "|"), want)
	}
}

// TestPipelineWithRunnel checks IBANs of a CSV file in a pipeline of Runnel (needs RUNNEL_BIN).
func TestPipelineWithRunnel(t *testing.T) {
	p := runneltest.NewProject(t, ".", "plugin.yaml")
	p.WriteFile("accounts.csv", "id,iban\n1,DE89 3704 0044 0532 0130 00\n2,DE89370400440532013001\n3,GB29NWBK60161331926819\n")
	p.WriteFile("pipelines/check.pipeline.yaml", `kind: Pipeline
apiVersion: runnel/v1
name: check
transforms:
  - id: read
    type: csv.input
    with:
      path: accounts.csv
      fields:
        - { name: id, type: int }
        - { name: iban, type: string }
  - id: check
    type: iban.validate
    with:
      field: iban
      bankCodeField: bank
  - id: good
    type: csv.output
    with:
      path: good.csv
      encoding: utf-8
      fields: [{ name: id }, { name: iban }, { name: bank }]
  - id: bad
    type: csv.output
    with:
      path: bad.csv
      encoding: utf-8
      fields: [{ name: id }, { name: _error }, { name: _errorField }]
links:
  - { from: read, to: check }
  - { from: check, to: good }
  - { from: check, to: bad, type: error }
`)
	if out, code := p.Validate(); code != 0 {
		t.Fatalf("validate: exit %d\n%s", code, out)
	}
	if out, code := p.Run("pipelines/check.pipeline.yaml"); code != 0 {
		t.Fatalf("run: exit %d\n%s", code, out)
	}
	good, _ := os.ReadFile(filepath.Join(p.Dir, "good.csv"))
	bad, _ := os.ReadFile(filepath.Join(p.Dir, "bad.csv"))
	if strings.ReplaceAll(string(good), "\r", "") != "id,iban,bank\n1,DE89370400440532013000,37040044\n3,GB29NWBK60161331926819,NWBK\n" {
		t.Errorf("good.csv:\n%s", good)
	}
	if !strings.Contains(string(bad), "2,invalid IBAN: the check digits are wrong") || !strings.Contains(string(bad), ",iban") {
		t.Errorf("bad.csv:\n%s", bad)
	}
	// init: a field that is not in the input stops the check.
	p.WriteFile("pipelines/check.pipeline.yaml", strings.Replace(readFile(t, filepath.Join(p.Dir, "pipelines", "check.pipeline.yaml")), "field: iban", "field: account", 1))
	if out, code := p.Validate(); code == 0 || !strings.Contains(out, `field "account" is not in the input`) {
		t.Fatalf("validate with wrong field: exit %d\n%s", code, out)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// BenchmarkValidateTransform checks 10,000 IBANs in batches of 500 through the protocol.
func BenchmarkValidateTransform(b *testing.B) {
	c := runneltest.Start(b, newPlugin())
	rows := make([][]any, 500)
	for i := range rows {
		rows[i] = []any{i, "DE89 3704 0044 0532 0130 00"}
	}
	batches := make([]runnel.Batch, 20)
	for i := range batches {
		batches[i] = runnel.Batch{Fields: accounts, Rows: rows}
	}
	b.ReportAllocs()
	for b.Loop() {
		if res, err := c.RunTransform("iban.validate", map[string]any{"field": "iban", "bankCodeField": "bank"}, batches...); err != nil || res.Error != "" {
			b.Fatal(err, res.Error)
		}
	}
	b.ReportMetric(float64(b.N*10000)/b.Elapsed().Seconds(), "records/s")
}
