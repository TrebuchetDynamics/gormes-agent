package memory

import (
	"context"
	"database/sql"

	"github.com/TrebuchetDynamics/gormes-agent/internal/memory/goncho"
)

const (
	GonchoMemoryV1ContractVersion = goncho.GonchoMemoryV1ContractVersion
	GonchoMemoryV1MarkdownFormat  = goncho.GonchoMemoryV1MarkdownFormat
	GonchoMemoryV1MCPToolContract = goncho.GonchoMemoryV1MCPToolContract
)

type GonchoMemoryV1ContractInfo = goncho.GonchoMemoryV1ContractInfo
type GonchoMemoryV1Status = goncho.GonchoMemoryV1Status
type GonchoMemoryV1Document = goncho.GonchoMemoryV1Document
type GonchoMemoryV1Item = goncho.GonchoMemoryV1Item
type GonchoMemoryV1RecallRequest = goncho.GonchoMemoryV1RecallRequest
type GonchoMemoryV1EvalArtifact = goncho.GonchoMemoryV1EvalArtifact

func GonchoMemoryV1Contract() GonchoMemoryV1ContractInfo {
	return goncho.GonchoMemoryV1Contract()
}

func ReadGonchoMemoryV1Status(ctx context.Context, db *sql.DB) (GonchoMemoryV1Status, error) {
	return goncho.ReadGonchoMemoryV1Status(ctx, db)
}

func ParseGonchoMemoryV1Markdown(body []byte) (GonchoMemoryV1Document, error) {
	return goncho.ParseGonchoMemoryV1Markdown(body)
}

func RenderGonchoMemoryV1Markdown(doc GonchoMemoryV1Document) (string, error) {
	return goncho.RenderGonchoMemoryV1Markdown(doc)
}

func ValidateGonchoMemoryV1Item(item GonchoMemoryV1Item) error {
	return goncho.ValidateGonchoMemoryV1Item(item)
}

func CanRecallGonchoMemoryV1(req GonchoMemoryV1RecallRequest, item GonchoMemoryV1Item) (bool, string) {
	return goncho.CanRecallGonchoMemoryV1(req, item)
}

func DecodeGonchoMemoryV1EvalArtifacts(body []byte) ([]GonchoMemoryV1EvalArtifact, error) {
	return goncho.DecodeGonchoMemoryV1EvalArtifacts(body)
}

func GonchoMemoryV1Checksum(content string) string {
	return goncho.GonchoMemoryV1Checksum(content)
}
