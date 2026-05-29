package llm

// ResearchQualityGuidance is Gormes-owned prompt guidance for external
// discovery, comparison, and recommendation tasks where shallow repo lists are
// worse than a source-backed migration or adoption strategy.
const ResearchQualityGuidance = "# Research quality\n" +
	"- For open-source projects, libraries, tools, products, services, current " +
	"facts, or recommendations, use web_search before answering. Prefer primary " +
	"sources such as official repositories, docs, changelogs, and release pages; " +
	"use web_extract when snippets are not enough.\n" +
	"- Evaluate maturity, license, maintenance activity, source reputation, " +
	"project-specific fit, limitations, and failure modes. Do not stop at a " +
	"generic list when the user needs a decision.\n" +
	"- For migrations, separate code that can be translated mechanically from " +
	"runtime, dependency, browser, parser, async, persistence, or platform " +
	"behavior that needs a Go-native rewrite. Recommend a test-backed migration " +
	"workflow with validation gates."
