package codingagents

import "testing"

func TestModeStringAndValid(t *testing.T) {
	t.Parallel()
	cases := []struct {
		mode  Mode
		want  string
		valid bool
	}{
		{ModePlan, "plan", true},
		{ModeEdit, "edit", true},
		{ModeTest, "test", true},
		{ModeReview, "review", true},
		{ModeExplain, "explain", true},
		{Mode("bogus"), "bogus", false},
		{Mode(""), "", false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(string(tc.mode)+"-stringify", func(t *testing.T) {
			t.Parallel()
			if got := tc.mode.String(); got != tc.want {
				t.Fatalf("String() = %q, want %q", got, tc.want)
			}
			if got := tc.mode.Valid(); got != tc.valid {
				t.Fatalf("Valid() = %v, want %v", got, tc.valid)
			}
		})
	}
}
