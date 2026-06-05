package memory

import "testing"

func TestRRFConfig_Defaults(t *testing.T) {
	cfg := RecallConfig{RRFEnabled: true}
	cfg.withDefaults()
	if cfg.RRFKFactor != 60 {
		t.Errorf("RRFKFactor = %v, want 60", cfg.RRFKFactor)
	}
	if cfg.RRFFTSWeight != 1.0 {
		t.Errorf("RRFFTSWeight = %v, want 1.0", cfg.RRFFTSWeight)
	}
	if cfg.RRFSemWeight != 1.0 {
		t.Errorf("RRFSemWeight = %v, want 1.0", cfg.RRFSemWeight)
	}
}

func TestRRFConfig_DisabledSkipsDefaults(t *testing.T) {
	cfg := RecallConfig{RRFEnabled: false}
	cfg.withDefaults()
	if cfg.RRFKFactor != 0 {
		t.Errorf("RRFKFactor should be 0 when disabled, got %v", cfg.RRFKFactor)
	}
}
