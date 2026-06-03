package main

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"

type hermesCLIParityStatus = gormescli.HermesCLIParityStatus

const (
	hermesCLIImplemented hermesCLIParityStatus = gormescli.HermesCLIImplemented
	hermesCLIRowBacked   hermesCLIParityStatus = gormescli.HermesCLIRowBacked
	hermesCLIOwned       hermesCLIParityStatus = gormescli.HermesCLIOwned
	hermesCLIExcluded    hermesCLIParityStatus = gormescli.HermesCLIExcluded
)

type hermesCLIParityKind = gormescli.HermesCLIParityKind

const (
	hermesCLICommand        hermesCLIParityKind = gormescli.HermesCLICommand
	hermesCLICommandSet     hermesCLIParityKind = gormescli.HermesCLICommandSet
	hermesCLIGlobalFlag     hermesCLIParityKind = gormescli.HermesCLIGlobalFlag
	hermesCLISlashCommand   hermesCLIParityKind = gormescli.HermesCLISlashCommand
	hermesCLIAlias          hermesCLIParityKind = gormescli.HermesCLIAlias
	hermesCLIGatewayHandler hermesCLIParityKind = gormescli.HermesCLIGatewayHandler
	hermesCLIPluginCommand  hermesCLIParityKind = gormescli.HermesCLIPluginCommand
)

type hermesCLIParityEntry = gormescli.HermesCLIParityEntry

func hermesCLIParityManifest() []hermesCLIParityEntry {
	return gormescli.HermesCLIParityManifest()
}
