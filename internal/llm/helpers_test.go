package llm

import (
	"os"
	"testing"
)

func TestIsTruthyValue_TrueCases(t *testing.T) {
	for _, val := range []interface{}{"1", "true", "yes", "on", "TRUE", "Yes", "ON", true} {
		if !IsTruthyValue(val, false) {
			t.Errorf("IsTruthyValue(%#v) = false, want true", val)
		}
	}
}

func TestIsTruthyValue_FalseCases(t *testing.T) {
	for _, val := range []interface{}{"0", "false", "no", "off", "maybe", "", false, nil} {
		if IsTruthyValue(val, false) {
			t.Errorf("IsTruthyValue(%#v) = true, want false", val)
		}
	}
}

func TestIsTruthyValue_NilReturnsDefault(t *testing.T) {
	if got := IsTruthyValue(nil, true); !got {
		t.Errorf("IsTruthyValue(nil, true) = false, want true (default)")
	}
	if got := IsTruthyValue(nil, false); got {
		t.Errorf("IsTruthyValue(nil, false) = true, want false (default)")
	}
}

func TestEnvVarEnabled_Truthy(t *testing.T) {
	t.Setenv("GORMES_TEST_TRUTHY", "true")
	if !EnvVarEnabled("GORMES_TEST_TRUTHY", "") {
		t.Error("EnvVarEnabled(GORMES_TEST_TRUTHY) = false, want true")
	}
}

func TestEnvVarEnabled_Falsy(t *testing.T) {
	t.Setenv("GORMES_TEST_FALSY", "false")
	if EnvVarEnabled("GORMES_TEST_FALSY", "") {
		t.Error("EnvVarEnabled(GORMES_TEST_FALSY) = true, want false")
	}
}

func TestEnvVarEnabled_Unset(t *testing.T) {
	if EnvVarEnabled("GORMES_TEST_NONEXISTENT_XYZ", "") {
		t.Error("EnvVarEnabled(nonexistent) = true, want false")
	}
}

func TestEnvVarEnabled_OnValue(t *testing.T) {
	t.Setenv("GORMES_TEST_ON", "on")
	if !EnvVarEnabled("GORMES_TEST_ON", "") {
		t.Error("EnvVarEnabled(GORMES_TEST_ON) = false, want true")
	}
}

func TestEnvVarEnabled_OneValue(t *testing.T) {
	t.Setenv("GORMES_TEST_ONE", "1")
	if !EnvVarEnabled("GORMES_TEST_ONE", "") {
		t.Error("EnvVarEnabled(GORMES_TEST_ONE) = false, want true")
	}
}

func TestIsTruthyValue_BoolTrueExplicit(t *testing.T) {
	if !IsTruthyValue(true, false) {
		t.Error("IsTruthyValue(true) = false, want true")
	}
}

func TestIsTruthyValue_BoolFalseExplicit(t *testing.T) {
	if IsTruthyValue(false, true) {
		t.Error("IsTruthyValue(false) = true, want false")
	}
}

// Cleanup env vars from previous tests.
func init() {
	os.Unsetenv("GORMES_TEST_TRUTHY")
	os.Unsetenv("GORMES_TEST_FALSY")
	os.Unsetenv("GORMES_TEST_ON")
	os.Unsetenv("GORMES_TEST_ONE")
}
