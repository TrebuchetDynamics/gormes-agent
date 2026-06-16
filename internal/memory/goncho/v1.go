package goncho

import (
	"context"
	"database/sql"

	memoryv1 "github.com/TrebuchetDynamics/gormes-agent/internal/memory/goncho/v1"
)

const (
	GonchoMemoryV1ContractVersion = memoryv1.GonchoMemoryV1ContractVersion
	GonchoMemoryV1MarkdownFormat  = memoryv1.GonchoMemoryV1MarkdownFormat
	GonchoMemoryV1MCPToolContract = memoryv1.GonchoMemoryV1MCPToolContract

	gonchoMemoryV1PrivateScope    = "private"
	gonchoMemoryV1SharedScope     = "shared"
	gonchoMemoryV1StateActive     = "active"
	gonchoMemoryV1StateTombstoned = "tombstoned"
)

type GonchoMemoryV1ContractInfo = memoryv1.GonchoMemoryV1ContractInfo
type GonchoMemoryV1Status = memoryv1.GonchoMemoryV1Status
type GonchoMemoryV1Document = memoryv1.GonchoMemoryV1Document
type GonchoMemoryV1Item = memoryv1.GonchoMemoryV1Item
type GonchoMemoryV1RecallRequest = memoryv1.GonchoMemoryV1RecallRequest
type GonchoMemoryV1EvalArtifact = memoryv1.GonchoMemoryV1EvalArtifact

func GonchoMemoryV1Contract() GonchoMemoryV1ContractInfo {
	return memoryv1.GonchoMemoryV1Contract()
}

func ReadGonchoMemoryV1Status(ctx context.Context, db *sql.DB) (GonchoMemoryV1Status, error) {
	return memoryv1.ReadGonchoMemoryV1Status(ctx, db)
}

func ParseGonchoMemoryV1Markdown(body []byte) (GonchoMemoryV1Document, error) {
	return memoryv1.ParseGonchoMemoryV1Markdown(body)
}

func RenderGonchoMemoryV1Markdown(doc GonchoMemoryV1Document) (string, error) {
	return memoryv1.RenderGonchoMemoryV1Markdown(doc)
}

func ValidateGonchoMemoryV1Item(item GonchoMemoryV1Item) error {
	return memoryv1.ValidateGonchoMemoryV1Item(item)
}

func CanRecallGonchoMemoryV1(req GonchoMemoryV1RecallRequest, item GonchoMemoryV1Item) (bool, string) {
	return memoryv1.CanRecallGonchoMemoryV1(req, item)
}

func DecodeGonchoMemoryV1EvalArtifacts(body []byte) ([]GonchoMemoryV1EvalArtifact, error) {
	return memoryv1.DecodeGonchoMemoryV1EvalArtifacts(body)
}

func GonchoMemoryV1Checksum(content string) string {
	return memoryv1.GonchoMemoryV1Checksum(content)
}
