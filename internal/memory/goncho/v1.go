package goncho

import (
	"bufio"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	GonchoMemoryV1ContractVersion    = "1"
	GonchoMemoryV1MarkdownFormat     = "1"
	GonchoMemoryV1MCPToolContract    = "1"
	gonchoMemoryV1MarkerStart        = "<!-- goncho-memory"
	gonchoMemoryV1MarkerEnd          = "-->"
	gonchoMemoryV1ClosingMarker      = "<!-- /goncho-memory -->"
	gonchoMemoryV1ForeignConfigReads = "denied"
	gonchoMemoryV1PrivateScope       = "private"
	gonchoMemoryV1SharedScope        = "shared"
	gonchoMemoryV1StateActive        = "active"
	gonchoMemoryV1StateTombstoned    = "tombstoned"
)

var gonchoMemoryV1Tables = []string{
	"goncho_memory_items",
	"goncho_memory_items_fts",
	"goncho_memory_eval_artifacts",
}

type GonchoMemoryV1ContractInfo struct {
	ContractVersion                string   `json:"contract_version"`
	MarkdownFormatVersion          string   `json:"markdown_format_version"`
	MCPToolContractVersion         string   `json:"mcp_tool_contract_version"`
	PrivateAgentMemoryDefault      bool     `json:"private_agent_memory_default"`
	SelfImprovementPerAgentDefault bool     `json:"self_improvement_per_agent_default"`
	ForeignConfigRuntimeReads      string   `json:"foreign_config_runtime_reads"`
	FastRecallPath                 []string `json:"fast_recall_path"`
	OptionalQualityLayers          []string `json:"optional_quality_layers"`
}

type GonchoMemoryV1Status struct {
	GonchoMemoryV1ContractInfo
	Tables map[string]bool `json:"tables"`
}

type GonchoMemoryV1Document struct {
	FormatVersion   string               `json:"format_version"`
	ContractVersion string               `json:"contract_version"`
	Items           []GonchoMemoryV1Item `json:"items"`
}

type GonchoMemoryV1Item struct {
	MemoryID        string   `json:"memory_id" yaml:"memory_id"`
	Revision        int      `json:"revision" yaml:"revision"`
	AgentID         string   `json:"agent_id" yaml:"agent_id"`
	WorkspaceID     string   `json:"workspace_id" yaml:"workspace_id"`
	PeerID          string   `json:"peer_id" yaml:"peer_id"`
	SessionID       string   `json:"session_id" yaml:"session_id"`
	Scope           string   `json:"scope" yaml:"scope"`
	State           string   `json:"state" yaml:"state"`
	SourceKind      string   `json:"source_kind" yaml:"source_kind"`
	SourceTurnID    string   `json:"source_turn_id,omitempty" yaml:"source_turn_id,omitempty"`
	TombstonedAt    string   `json:"tombstoned_at,omitempty" yaml:"tombstoned_at,omitempty"`
	TombstoneReason string   `json:"tombstone_reason,omitempty" yaml:"tombstone_reason,omitempty"`
	Checksum        string   `json:"checksum" yaml:"checksum"`
	Tags            []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Importance      float64  `json:"importance" yaml:"importance"`
	CreatedAt       string   `json:"created_at" yaml:"created_at"`
	UpdatedAt       string   `json:"updated_at" yaml:"updated_at"`
	ProvenanceJSON  string   `json:"provenance_json,omitempty" yaml:"provenance_json,omitempty"`
	Content         string   `json:"content" yaml:"-"`
}

type GonchoMemoryV1RecallRequest struct {
	AgentID     string
	WorkspaceID string
	AllowShared bool
}

type GonchoMemoryV1EvalArtifact struct {
	ArtifactID     string         `json:"artifact_id"`
	AgentID        string         `json:"agent_id"`
	WorkspaceID    string         `json:"workspace_id"`
	PeerID         string         `json:"peer_id"`
	SessionID      string         `json:"session_id"`
	Type           string         `json:"type"`
	Status         string         `json:"status"`
	SourceMemoryID string         `json:"source_memory_id"`
	Shared         bool           `json:"shared"`
	Payload        map[string]any `json:"payload"`
}

func GonchoMemoryV1Contract() GonchoMemoryV1ContractInfo {
	return GonchoMemoryV1ContractInfo{
		ContractVersion:                GonchoMemoryV1ContractVersion,
		MarkdownFormatVersion:          GonchoMemoryV1MarkdownFormat,
		MCPToolContractVersion:         GonchoMemoryV1MCPToolContract,
		PrivateAgentMemoryDefault:      true,
		SelfImprovementPerAgentDefault: true,
		ForeignConfigRuntimeReads:      gonchoMemoryV1ForeignConfigReads,
		FastRecallPath:                 []string{"sqlite", "fts5", "graph"},
		OptionalQualityLayers:          []string{"embeddings", "qmd_deep_search", "dialectic", "dream_consolidation"},
	}
}

func ReadGonchoMemoryV1Status(ctx context.Context, db *sql.DB) (GonchoMemoryV1Status, error) {
	if db == nil {
		return GonchoMemoryV1Status{}, errors.New("memory: nil db")
	}
	status := GonchoMemoryV1Status{
		GonchoMemoryV1ContractInfo: GonchoMemoryV1Contract(),
		Tables:                     make(map[string]bool, len(gonchoMemoryV1Tables)),
	}
	for _, table := range gonchoMemoryV1Tables {
		status.Tables[table] = false
	}
	rows, err := db.QueryContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table'`)
	if err != nil {
		return GonchoMemoryV1Status{}, fmt.Errorf("memory: goncho v1 tables: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return GonchoMemoryV1Status{}, fmt.Errorf("memory: scan goncho v1 table: %w", err)
		}
		if _, ok := status.Tables[name]; ok {
			status.Tables[name] = true
		}
	}
	if err := rows.Err(); err != nil {
		return GonchoMemoryV1Status{}, fmt.Errorf("memory: goncho v1 table rows: %w", err)
	}
	return status, nil
}

func ParseGonchoMemoryV1Markdown(body []byte) (GonchoMemoryV1Document, error) {
	text := string(body)
	header, rest, err := parseGonchoMemoryV1FrontMatter(text)
	if err != nil {
		return GonchoMemoryV1Document{}, err
	}
	doc := GonchoMemoryV1Document{
		FormatVersion:   header.FormatVersion,
		ContractVersion: header.ContractVersion,
	}
	for {
		start := strings.Index(rest, gonchoMemoryV1MarkerStart)
		if start < 0 {
			break
		}
		afterStart := rest[start+len(gonchoMemoryV1MarkerStart):]
		metaEnd := strings.Index(afterStart, gonchoMemoryV1MarkerEnd)
		if metaEnd < 0 {
			return GonchoMemoryV1Document{}, errors.New("memory: unterminated goncho-memory metadata block")
		}
		metaRaw := strings.TrimSpace(afterStart[:metaEnd])
		afterMeta := afterStart[metaEnd+len(gonchoMemoryV1MarkerEnd):]
		contentEnd := strings.Index(afterMeta, gonchoMemoryV1ClosingMarker)
		if contentEnd < 0 {
			return GonchoMemoryV1Document{}, errors.New("memory: unterminated goncho-memory content block")
		}
		content := strings.Trim(afterMeta[:contentEnd], "\n")
		var item GonchoMemoryV1Item
		if err := yaml.Unmarshal([]byte(metaRaw), &item); err != nil {
			return GonchoMemoryV1Document{}, fmt.Errorf("memory: parse goncho-memory metadata: %w", err)
		}
		item.Content = content
		doc.Items = append(doc.Items, item)
		rest = afterMeta[contentEnd+len(gonchoMemoryV1ClosingMarker):]
	}
	return doc, nil
}

func RenderGonchoMemoryV1Markdown(doc GonchoMemoryV1Document) (string, error) {
	if doc.FormatVersion == "" {
		doc.FormatVersion = GonchoMemoryV1MarkdownFormat
	}
	if doc.ContractVersion == "" {
		doc.ContractVersion = GonchoMemoryV1ContractVersion
	}
	items := append([]GonchoMemoryV1Item(nil), doc.Items...)
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].MemoryID < items[j].MemoryID
	})
	var b strings.Builder
	fmt.Fprintf(&b, "---\ngoncho_memory_format: %q\ngoncho_memory_contract: %q\n---\n\n", doc.FormatVersion, doc.ContractVersion)
	b.WriteString("# Goncho Memory V1 Export\n\n")
	for _, item := range items {
		if item.Checksum == "" {
			item.Checksum = GonchoMemoryV1Checksum(item.Content)
		}
		meta := item
		meta.Content = ""
		raw, err := yaml.Marshal(meta)
		if err != nil {
			return "", fmt.Errorf("memory: render goncho-memory metadata: %w", err)
		}
		b.WriteString(gonchoMemoryV1MarkerStart)
		b.WriteByte('\n')
		b.Write(raw)
		b.WriteString(gonchoMemoryV1MarkerEnd)
		b.WriteByte('\n')
		b.WriteString(strings.Trim(item.Content, "\n"))
		b.WriteByte('\n')
		b.WriteString(gonchoMemoryV1ClosingMarker)
		b.WriteString("\n\n")
	}
	return b.String(), nil
}

func ValidateGonchoMemoryV1Item(item GonchoMemoryV1Item) error {
	switch {
	case strings.TrimSpace(item.MemoryID) == "":
		return errors.New("memory: memory_id is required")
	case strings.TrimSpace(item.AgentID) == "":
		return errors.New("memory: agent_id is required")
	case strings.TrimSpace(item.WorkspaceID) == "":
		return errors.New("memory: workspace_id is required")
	case strings.TrimSpace(item.PeerID) == "":
		return errors.New("memory: peer_id is required")
	case item.Revision <= 0:
		return errors.New("memory: revision must be positive")
	case item.Scope != gonchoMemoryV1PrivateScope && item.Scope != gonchoMemoryV1SharedScope:
		return fmt.Errorf("memory: unsupported scope %q", item.Scope)
	case item.State != gonchoMemoryV1StateActive && item.State != gonchoMemoryV1StateTombstoned:
		return fmt.Errorf("memory: unsupported state %q", item.State)
	case item.State == gonchoMemoryV1StateTombstoned && strings.TrimSpace(item.TombstonedAt) == "":
		return errors.New("memory: tombstoned memories require tombstoned_at")
	case strings.TrimSpace(item.Content) == "":
		return errors.New("memory: content is required")
	}
	if item.Checksum != "" && item.Checksum != GonchoMemoryV1Checksum(item.Content) {
		return fmt.Errorf("memory: checksum mismatch for %s", item.MemoryID)
	}
	return nil
}

func CanRecallGonchoMemoryV1(req GonchoMemoryV1RecallRequest, item GonchoMemoryV1Item) (bool, string) {
	if item.State == gonchoMemoryV1StateTombstoned {
		return false, "tombstoned"
	}
	if item.AgentID == req.AgentID && item.WorkspaceID == req.WorkspaceID {
		return true, "owner_agent"
	}
	if req.AllowShared && item.Scope == gonchoMemoryV1SharedScope && item.WorkspaceID == req.WorkspaceID {
		return true, "shared_workspace"
	}
	return false, "private_agent_boundary"
}

func DecodeGonchoMemoryV1EvalArtifacts(body []byte) ([]GonchoMemoryV1EvalArtifact, error) {
	scanner := bufio.NewScanner(strings.NewReader(string(body)))
	var out []GonchoMemoryV1EvalArtifact
	for lineNo := 1; scanner.Scan(); lineNo++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var artifact GonchoMemoryV1EvalArtifact
		if err := json.Unmarshal([]byte(line), &artifact); err != nil {
			return nil, fmt.Errorf("memory: parse eval artifact line %d: %w", lineNo, err)
		}
		if strings.TrimSpace(artifact.AgentID) == "" || strings.TrimSpace(artifact.WorkspaceID) == "" {
			return nil, fmt.Errorf("memory: eval artifact line %d missing agent/workspace scope", lineNo)
		}
		out = append(out, artifact)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("memory: scan eval artifacts: %w", err)
	}
	return out, nil
}

func GonchoMemoryV1Checksum(content string) string {
	sum := sha256.Sum256([]byte(strings.Trim(content, "\n")))
	return "sha256:" + hex.EncodeToString(sum[:])
}

type gonchoMemoryV1FrontMatter struct {
	FormatVersion   string `yaml:"goncho_memory_format"`
	ContractVersion string `yaml:"goncho_memory_contract"`
}

func parseGonchoMemoryV1FrontMatter(text string) (gonchoMemoryV1FrontMatter, string, error) {
	if !strings.HasPrefix(text, "---\n") {
		return gonchoMemoryV1FrontMatter{}, "", errors.New("memory: goncho memory markdown missing frontmatter")
	}
	rest := text[len("---\n"):]
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		return gonchoMemoryV1FrontMatter{}, "", errors.New("memory: goncho memory markdown unterminated frontmatter")
	}
	var fm gonchoMemoryV1FrontMatter
	if err := yaml.Unmarshal([]byte(rest[:end]), &fm); err != nil {
		return gonchoMemoryV1FrontMatter{}, "", fmt.Errorf("memory: parse goncho memory frontmatter: %w", err)
	}
	if fm.FormatVersion == "" || fm.ContractVersion == "" {
		return gonchoMemoryV1FrontMatter{}, "", errors.New("memory: goncho memory frontmatter missing format or contract version")
	}
	return fm, rest[end+len("\n---\n"):], nil
}
