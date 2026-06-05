package gormescli

import "github.com/TrebuchetDynamics/gormes-agent/internal/app/setupprofile"

func SetupProfileID(home, defaultProfileID string) string {
	return setupprofile.ProfileID(home, defaultProfileID)
}

func SetupProfileRegistryPath(baseHome string) string {
	return setupprofile.RegistryPath(baseHome)
}

func SetupProfileCredentialID(profileID, channelID string) string {
	return setupprofile.CredentialID(profileID, channelID)
}

func CompactSetupProfileStrings(values []string) []string {
	return setupprofile.CompactStrings(values)
}

func SetupInt64Strings(values []int64) []string {
	return setupprofile.Int64Strings(values)
}
