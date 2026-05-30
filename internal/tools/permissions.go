package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/permissions"

type PermissionManifest = permissions.PermissionManifest
type DirectoryScope = permissions.DirectoryScope
type ApprovalRecord = permissions.ApprovalRecord

func DefaultPermissionManifestPath() string {
	return permissions.DefaultPermissionManifestPath()
}

func LoadPermissionManifest(path string) (*PermissionManifest, error) {
	return permissions.LoadPermissionManifest(path)
}

func CheckPathScope(path, operation string, scopes []DirectoryScope) error {
	return permissions.CheckPathScope(path, operation, scopes)
}

func CheckCwdOnly(workdir, cwd string) error {
	return permissions.CheckCwdOnly(workdir, cwd)
}

func FindApprovalRecord(records []ApprovalRecord, cmd string) (ApprovalRecord, bool) {
	return permissions.FindApprovalRecord(records, cmd)
}

func GetApprovalManifest() (*PermissionManifest, error) {
	return permissions.GetApprovalManifest()
}

func ResetApprovalManifestCache() {
	permissions.ResetApprovalManifestCache()
}
