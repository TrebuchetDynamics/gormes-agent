package auxiliary

import "testing"

func TestNormalizeTaskDefaultCurator(t *testing.T) {
	task := AuxiliaryTaskCfg{Provider: " auto ", Model: " model ", BaseURL: " http://localhost ", APIKey: " key ", Timeout: 0}

	NormalizeTask(&task, true)

	if task.Provider != "auto" || task.Model != "model" || task.BaseURL != "http://localhost" || task.APIKey != "key" {
		t.Fatalf("task = %+v, want trimmed fields", task)
	}
	if task.Timeout != 600 {
		t.Fatalf("Timeout = %d, want curator default 600", task.Timeout)
	}
	if task.ExtraBody == nil {
		t.Fatal("ExtraBody = nil, want empty map")
	}
}

func TestNormalizeTaskNonCuratorOnlyTrims(t *testing.T) {
	task := AuxiliaryTaskCfg{Provider: " custom ", Model: " mini ", Timeout: 0}

	NormalizeTask(&task, false)

	if task.Provider != "custom" || task.Model != "mini" {
		t.Fatalf("task = %+v, want trimmed fields", task)
	}
	if task.Timeout != 0 || task.ExtraBody != nil {
		t.Fatalf("non-curator defaults = timeout %d extra %#v, want unchanged", task.Timeout, task.ExtraBody)
	}
}
