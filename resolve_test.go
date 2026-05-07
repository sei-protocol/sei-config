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
	if got := err.Error(); got != "unsupported slice element type: int" {
		t.Errorf("got %q, want %q", got, "unsupported slice element type: int")
	}
}

func TestParseStringSlice(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", []string{}},
		{"a", []string{"a"}},
		{"a,b,c", []string{"a", "b", "c"}},
		{" a , b , c ", []string{"a", "b", "c"}},
		{",a,,b,", []string{"a", "b"}},
		{",,,", []string{}},
	}
	for _, tc := range cases {
		got := parseStringSlice(tc.in)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("parseStringSlice(%q): got %v, want %v", tc.in, got, tc.want)
		}
		if got == nil {
			t.Errorf("parseStringSlice(%q) returned nil; want non-nil empty slice", tc.in)
		}
	}
}
