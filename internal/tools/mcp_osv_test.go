package tools

import (
	"context"
	"testing"
)

type fakeRootOSVClient struct{}

func (fakeRootOSVClient) Query(context.Context, OSVPackageQuery) ([]OSVVulnerability, error) {
	return []OSVVulnerability{{ID: "MAL-2026-root", Summary: "malware"}}, nil
}

func TestMCPPackageLaunchOSVFacade(t *testing.T) {
	res := CheckMCPServerPackageLaunch(context.Background(), MCPServerDefinition{
		Name:    "pkg",
		Command: "npx",
		Args:    []string{"bad-package"},
	}, fakeRootOSVClient{})
	if !res.Blocked || res.Evidence != OSVEvidenceMalwareFound {
		t.Fatalf("res = %+v, want malware block", res)
	}
}
