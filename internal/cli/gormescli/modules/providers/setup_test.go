package providers

import (
	"reflect"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli/gormescli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/progress"
)

func TestSetupSectionsDeclareProviderOwnership(t *testing.T) {
	got := SetupSections()
	want := []gormescli.SetupSection{
		{Name: "provider", Label: "Provider", Module: progress.ModuleProviders},
		{Name: "model", Label: "Model", Module: progress.ModuleProviders},
		{Name: "fallback", Label: "Fallback Providers", Module: progress.ModuleProviders},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SetupSections() = %#v, want %#v", got, want)
	}
}
