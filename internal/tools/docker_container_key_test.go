package tools

import "testing"

func TestDockerContainerKey_TopLevelDefault(t *testing.T) {
	got := DockerContainerKey(DockerContainerRequest{})
	if got != "default" {
		t.Fatalf("DockerContainerKey(empty top-level) = %q, want default", got)
	}
}

func TestDockerContainerKey_TopLevelExplicitTaskID(t *testing.T) {
	got := DockerContainerKey(DockerContainerRequest{TaskID: "manual"})
	if got != "manual" {
		t.Fatalf("DockerContainerKey(top-level explicit task ID) = %q, want manual", got)
	}
}

func TestDockerContainerKey_SubagentRequiresIsolatedTaskID(t *testing.T) {
	tests := []struct {
		name string
		req  DockerContainerRequest
		want string
	}{
		{
			name: "missing task ID",
			req:  DockerContainerRequest{IsSubagent: true},
			want: "",
		},
		{
			name: "explicit task ID",
			req:  DockerContainerRequest{TaskID: "subagent-1", IsSubagent: true},
			want: "subagent-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DockerContainerKey(tt.req)
			if got != tt.want {
				t.Fatalf("DockerContainerKey(%+v) = %q, want %q", tt.req, got, tt.want)
			}
		})
	}
}

func TestDockerContainerKey_RolloutRequiresIsolatedTaskID(t *testing.T) {
	tests := []struct {
		name string
		req  DockerContainerRequest
		want string
	}{
		{
			name: "missing task ID",
			req:  DockerContainerRequest{IsRollout: true},
			want: "",
		},
		{
			name: "explicit task ID",
			req:  DockerContainerRequest{TaskID: "rollout-1", IsRollout: true},
			want: "rollout-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DockerContainerKey(tt.req)
			if got != tt.want {
				t.Fatalf("DockerContainerKey(%+v) = %q, want %q", tt.req, got, tt.want)
			}
		})
	}
}

func TestDockerContainerKey_TrimsWhitespace(t *testing.T) {
	tests := []struct {
		name string
		req  DockerContainerRequest
		want string
	}{
		{
			name: "top-level default",
			req:  DockerContainerRequest{TaskID: " \t\n "},
			want: "default",
		},
		{
			name: "top-level explicit",
			req:  DockerContainerRequest{TaskID: " manual \n"},
			want: "manual",
		},
		{
			name: "subagent missing",
			req:  DockerContainerRequest{TaskID: " \t", IsSubagent: true},
			want: "",
		},
		{
			name: "subagent explicit",
			req:  DockerContainerRequest{TaskID: "\tsubagent-1\n", IsSubagent: true},
			want: "subagent-1",
		},
		{
			name: "rollout missing",
			req:  DockerContainerRequest{TaskID: "\n", IsRollout: true},
			want: "",
		},
		{
			name: "rollout explicit",
			req:  DockerContainerRequest{TaskID: " rollout-1 ", IsRollout: true},
			want: "rollout-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DockerContainerKey(tt.req)
			if got != tt.want {
				t.Fatalf("DockerContainerKey(%+v) = %q, want %q", tt.req, got, tt.want)
			}
		})
	}
}
