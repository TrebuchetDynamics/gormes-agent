package environment

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/environment/credentialfiles"

// CredentialFilesMount describes one host→container file mount produced by
// the credential-files registry.
type CredentialFilesMount = credentialfiles.Mount

// CredentialFilesRegistry is a session-scoped registry of host credential
// files that must be mounted into remote sandbox environments (Docker, SSH,
// Modal). It mirrors Hermes' tools/credential_files.py session registry.
type CredentialFilesRegistry = credentialfiles.Registry

// NewCredentialFilesRegistry creates a Registry rooted at gormesHome and
// pre-registers any configuredPaths entries (from config.terminal.credential_files).
func NewCredentialFilesRegistry(gormesHome string, configuredPaths []string) *CredentialFilesRegistry {
	return credentialfiles.NewRegistry(gormesHome, configuredPaths)
}

// SkillsDirectoryMount returns the mount that maps the GORMES_HOME skills
// directory into a container. Returns nil when the skills directory is absent.
// containerBase defaults to "/root/.gormes" when empty.
func SkillsDirectoryMount(gormesHome, containerBase string) *CredentialFilesMount {
	return credentialfiles.SkillsDirectoryMount(gormesHome, containerBase)
}

// IterSkillsFiles calls fn for each file under the GORMES_HOME skills
// directory, passing the host path and the container-absolute path.
// containerBase defaults to "/root/.gormes" when empty.
func IterSkillsFiles(gormesHome, containerBase string, fn func(hostPath, containerPath string)) error {
	return credentialfiles.IterSkillsFiles(gormesHome, containerBase, fn)
}
