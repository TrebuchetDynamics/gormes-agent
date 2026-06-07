package subprocess

import (
	"reflect"
	"testing"
)

func TestEnvWithHomeReplacesFirstHomeAndDropsDuplicates(t *testing.T) {
	got := EnvWithHome([]string{"PATH=/bin", "HOME=/old", "HOME=/older", "SHELL=/bin/sh"}, func() (string, bool) {
		return " /profile/home ", true
	})
	want := []string{"PATH=/bin", "HOME=/profile/home", "SHELL=/bin/sh"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("EnvWithHome = %#v, want %#v", got, want)
	}
}

func TestEnvWithHomePreservesEnvWhenResolverEmpty(t *testing.T) {
	input := []string{"PATH=/bin", "HOME=/old"}
	got := EnvWithHome(input, func() (string, bool) { return " ", true })
	if !reflect.DeepEqual(got, input) {
		t.Fatalf("EnvWithHome = %#v, want original %#v", got, input)
	}
	got[0] = "PATH=/changed"
	if input[0] != "PATH=/bin" {
		t.Fatalf("returned env should not alias input: %#v", input)
	}
}
