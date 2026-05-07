package seiconfig

import (
	"reflect"
	"testing"
)

func TestSetReflectValue_StringSlice(t *testing.T) {
	var s []string
	v := reflect.ValueOf(&s).Elem()

	if err := setReflectValue(v, "a, b ,c"); err != nil {
		t.Fatalf("setReflectValue: %v", err)
	}
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(s, want) {
		t.Errorf("got %v, want %v", s, want)
	}
}

func TestSetReflectValue_RejectsNonStringSlice(t *testing.T) {
	var s []int
	v := reflect.ValueOf(&s).Elem()

	err := setReflectValue(v, "1,2,3")
	if err == nil {
		t.Fatal("expected error for []int slice")
	}
	if got := err.Error(); got != "unsupported slice element kind: int" {
		t.Errorf("got %q, want %q", got, "unsupported slice element kind: int")
	}
}

func TestSetReflectValue_RejectsSliceOfSlice(t *testing.T) {
	var s [][]string
	v := reflect.ValueOf(&s).Elem()

	err := setReflectValue(v, "anything")
	if err == nil {
		t.Fatal("expected error for [][]string")
	}
	if got := err.Error(); got != "unsupported slice element kind: slice" {
		t.Errorf("got %q, want %q", got, "unsupported slice element kind: slice")
	}
}

func TestParseStringSlice(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"empty yields non-nil empty", "", []string{}},
		{"single value", "a", []string{"a"}},
		{"multi value", "a,b,c", []string{"a", "b", "c"}},
		{"trims whitespace", " a , b , c ", []string{"a", "b", "c"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseStringSlice(tc.in)
			if err != nil {
				t.Fatalf("parseStringSlice(%q): %v", tc.in, err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("parseStringSlice(%q): got %v, want %v", tc.in, got, tc.want)
			}
			if got == nil {
				t.Errorf("parseStringSlice(%q) returned nil; want non-nil empty slice", tc.in)
			}
		})
	}
}

func TestParseStringSlice_RejectsEmptyEntries(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"leading comma", ",a"},
		{"trailing comma", "a,"},
		{"consecutive commas", "a,,b"},
		{"only whitespace entry", "a, ,b"},
		{"only commas", ",,,"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseStringSlice(tc.in)
			if err == nil {
				t.Fatalf("parseStringSlice(%q): expected error, got nil", tc.in)
			}
		})
	}
}
