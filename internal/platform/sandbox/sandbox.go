package sandbox

// Sandbox represents an isolated execution environment for a session.
type Sandbox interface {
	SessionID() string
	WorkspaceDir() string
	UploadsDir() string
	OutputsDir() string
}

type sandbox struct {
	sessionID    string
	workspaceDir string
	uploadsDir   string
	outputsDir   string
}

func (s *sandbox) SessionID() string    { return s.sessionID }
func (s *sandbox) WorkspaceDir() string { return s.workspaceDir }
func (s *sandbox) UploadsDir() string   { return s.uploadsDir }
func (s *sandbox) OutputsDir() string   { return s.outputsDir }
