package site

type NavLink struct {
	Label string
	Href  string
}

type Link struct {
	Label string
	Href  string
}

type ProofStat struct {
	Label string
	Value string
	Tone  string
}

type CommandStep struct {
	Label string
	Note  string
	Lines []string
}

type OpsModule struct {
	Label string
	Title string
	Body  string
}

type ShipState struct {
	State string
	Name  string
	Body  string
}

type LandingPage struct {
	Title            string
	Description      string
	Nav              []NavLink
	HeroKicker       string
	HeroHeadline     string
	HeroCopy         []string
	HeroPanelTitle   string
	HeroPanelLines   []string
	PrimaryCTA       Link
	SecondaryCTA     Link
	TertiaryCTA      Link
	ScopeNote        string
	ProofStats       []ProofStat
	ActivationTitle  string
	ActivationIntro  string
	ActivationSteps  []CommandStep
	OpsTitle         string
	OpsIntro         string
	OpsModules       []OpsModule
	RoadmapTitle     string
	RoadmapIntro     string
	ShipStates       []ShipState
	ContributorTitle string
	ContributorBody  string
	ContributorLinks []Link
	FooterLinks      []Link
	FooterLine       string
}

func DefaultPage() LandingPage {
	return LandingPage{
		Title:       "Gormes.ai | Run Hermes Through a Go Operator Console",
		Description: "Gormes is the Go operator shell for Hermes users: zero-CGO, Go-native tools, split Telegram edge, and honest shipping state.",
		Nav: []NavLink{
			{Label: "Run Now", Href: "#quickstart"},
			{Label: "Shipping State", Href: "#roadmap"},
			{Label: "Source", Href: "#contribute"},
			{Label: "GitHub", Href: "https://github.com/XelHaku/gormes-agent"},
		},
		HeroKicker:   "HERMES / GO OPERATOR SHELL",
		HeroHeadline: "Run Hermes Through a Go Operator Console.",
		HeroCopy: []string{
			"Stop waiting for the clean-room rewrite. Gormes already ships a Go shell, a Go-native tool loop, Route-B resilience, and a split Telegram edge.",
			"Boot it locally. Judge the surface yourself. Keep the promises honest.",
		},
		HeroPanelTitle: "Boot Sequence",
		HeroPanelLines: []string{
			"[ok] shell compiled",
			"[ok] tool loop armed",
			"[ok] route-b ready",
			"[warn] transcript memory still on later cutover path",
		},
		PrimaryCTA:   Link{Label: "Boot Gormes", Href: "#quickstart"},
		SecondaryCTA: Link{Label: "See Shipping State", Href: "#roadmap"},
		TertiaryCTA:  Link{Label: "Inspect Source", Href: "https://github.com/XelHaku/gormes-agent"},
		ScopeNote:    "Current boundary: the Go shell ships now. Transcript memory stays on the later cutover path.",
		ProofStats: []ProofStat{
			{Label: "gormes shell", Value: "8.2M shell", Tone: "live"},
			{Label: "telegram edge", Value: "15M telegram edge", Tone: "cold"},
			{Label: "deployment", Value: "Zero-CGO", Tone: "live"},
			{Label: "tool surface", Value: "Go-native", Tone: "cold"},
			{Label: "phase", Value: "2 ships now", Tone: "warn"},
		},
		ActivationTitle: "Run the shell. Judge it yourself.",
		ActivationIntro: "If you already trust Hermes, this is the shortest honest path to feel the Go layer.",
		ActivationSteps: []CommandStep{
			{
				Label: "01 / BACKEND",
				Note:  "Bring up the existing Hermes backend first.",
				Lines: []string{"API_SERVER_ENABLED=true hermes gateway start"},
			},
			{
				Label: "02 / BUILD",
				Note:  "Compile the Go surfaces with the current local toolchain.",
				Lines: []string{"cd gormes", "make build"},
			},
			{
				Label: "03 / VERIFY",
				Note:  "Run doctor, launch the shell, and probe the Telegram edge if needed.",
				Lines: []string{"./bin/gormes doctor --offline", "./bin/gormes", "GORMES_TELEGRAM_TOKEN=... GORMES_TELEGRAM_CHAT_ID=123456789 ./bin/gormes-telegram"},
			},
		},
		OpsTitle: "Why Hermes users switch",
		OpsIntro: "Gormes is not a reskin. It is the hardened shell around the workflows you already trust.",
		OpsModules: []OpsModule{
			{Label: "RESPONSIVENESS", Title: "Cut startup tax", Body: "Use the Go shell that boots like a tool, not a ceremony."},
			{Label: "TOOLS", Title: "Keep the loop typed", Body: "Run the Go-native tool surface in-process and verify it before you spend more tokens."},
			{Label: "ISOLATION", Title: "Split the blast radius", Body: "Keep Telegram and the shell in separate binaries so dependencies and failures stay local."},
			{Label: "HONESTY", Title: "Ship the boundary you have", Body: "The shell is real now. Transcript memory and the brain cutover are still later work."},
		},
		RoadmapTitle: "Shipping State, Not Wishcasting",
		RoadmapIntro: "This ledger separates what already ships on trunk from the next real handoff lines.",
		ShipStates: []ShipState{
			{State: "SHIPPED", Name: "Phase 1 — Dashboard", Body: "The Bubble Tea shell and operator surface are already real."},
			{State: "SHIPPED", Name: "Phase 2 — Gateway", Body: "Tool registry, Telegram scout, and thin session resume already live on trunk."},
			{State: "NEXT", Name: "Phase 3 — Memory", Body: "SQLite + FTS5 transcript memory still marks the real handoff line."},
			{State: "LATER", Name: "Phase 4 — Brain", Body: "Prompt building and native agent orchestration move after memory is real."},
		},
		ContributorTitle: "Inspect the Machine",
		ContributorBody:  "Read the architecture, inspect the source, and verify the claims like an operator, not a spectator.",
		ContributorLinks: []Link{
			{Label: "Read ARCH_PLAN.md", Href: "https://github.com/XelHaku/gormes-agent/blob/main/gormes/docs/ARCH_PLAN.md"},
			{Label: "Browse the Gormes source", Href: "https://github.com/XelHaku/gormes-agent/tree/main/gormes"},
			{Label: "Open the implementation docs", Href: "https://github.com/XelHaku/gormes-agent/tree/main/gormes/docs/superpowers"},
		},
		FooterLinks: []Link{
			{Label: "GitHub", Href: "https://github.com/XelHaku/gormes-agent"},
			{Label: "ARCH_PLAN", Href: "https://github.com/XelHaku/gormes-agent/blob/main/gormes/docs/ARCH_PLAN.md"},
			{Label: "Hermes Upstream", Href: "https://github.com/NousResearch/hermes-agent"},
			{Label: "MIT License", Href: "https://github.com/XelHaku/gormes-agent/blob/main/LICENSE"},
		},
		FooterLine: "Gormes already ships the operator shell. The memory lattice and brain cutover come later.",
	}
}
