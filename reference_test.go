package main

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/runnelhq/sdk-go/runneltest"
)

// TestReference checks the reference of every type with the rules of Runnel.
func TestReference(t *testing.T) {
	runneltest.CheckReference(t, newPlugin())
}

// TestReferenceOutputs checks that the reference of each action names exactly the fields
// of its result struct, so that both change together.
func TestReferenceOutputs(t *testing.T) {
	results := map[string]any{"iban.validate": ibanOutput{}, "iban.validate-bic": bicOutput{}, "iban.create": ibanOutput{}}
	raw, err := runneltest.Start(t, newPlugin()).Call("describe", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	var d struct {
		Actions []struct {
			Type    string `json:"type"`
			Outputs []struct {
				Name string `json:"name"`
				Type string `json:"type"`
			} `json:"outputs"`
		} `json:"actions"`
	}
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	for _, a := range d.Actions {
		rt := reflect.TypeOf(results[a.Type])
		var want, got []string
		for f := range rt.Fields() {
			typ := "string"
			if f.Type.Kind() == reflect.Bool {
				typ = "boolean"
			}
			name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
			want = append(want, "output."+name+" "+typ)
		}
		for _, o := range a.Outputs {
			got = append(got, o.Name+" "+o.Type)
		}
		if !slices.Equal(got, want) {
			t.Errorf("%s: outputs\n%v\nwant (from the result struct)\n%v", a.Type, got, want)
		}
	}
}

// TestReferenceTransformOutputs checks that the transform describes the new fields of its
// configuration.
func TestReferenceTransformOutputs(t *testing.T) {
	var names []string
	for _, o := range validateFieldReference.Outputs {
		names = append(names, o.Name)
	}
	want := []string{"<field>"}
	for _, f := range (validateFieldConfig{CountryField: "a", BankCodeField: "b", BranchCodeField: "c"}).addedFields() {
		want = append(want, "<"+f[0]+">")
	}
	if !slices.Equal(names, want) {
		t.Errorf("outputs %v, want %v", names, want)
	}
}
