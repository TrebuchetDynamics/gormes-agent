package subagent

import (
	"context"
	"testing"
)

func TestBackgroundReviewProvenance(t *testing.T) {
	if got := SkillWriteOrigin(context.Background()); got != SkillWriteOriginForeground {
		t.Fatalf("default origin = %q, want %q", got, SkillWriteOriginForeground)
	}
	if IsBackgroundReviewSkillWrite(context.Background()) {
		t.Fatal("background review detected on empty context")
	}

	ctx := WithSkillWriteOrigin(context.Background(), SkillWriteOriginBackgroundReview)
	if got := SkillWriteOrigin(ctx); got != SkillWriteOriginBackgroundReview {
		t.Fatalf("origin = %q, want %q", got, SkillWriteOriginBackgroundReview)
	}
	if !IsBackgroundReviewSkillWrite(ctx) {
		t.Fatal("background review origin not detected")
	}

	if got := SkillWriteOrigin(WithSkillWriteOrigin(context.Background(), "")); got != SkillWriteOriginForeground {
		t.Fatalf("empty origin = %q, want foreground", got)
	}
}
