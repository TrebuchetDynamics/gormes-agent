package gormescli

import appparity "github.com/TrebuchetDynamics/gormes-agent/internal/app/hermescliparity"

type HermesCLIParityStatus = appparity.Status

const (
	HermesCLIImplemented HermesCLIParityStatus = appparity.StatusImplemented
	HermesCLIRowBacked   HermesCLIParityStatus = appparity.StatusRowBacked
	HermesCLIOwned       HermesCLIParityStatus = appparity.StatusOwned
	HermesCLIExcluded    HermesCLIParityStatus = appparity.StatusExcluded
)

type HermesCLIParityKind = appparity.Kind

const (
	HermesCLICommand        HermesCLIParityKind = appparity.KindCommand
	HermesCLICommandSet     HermesCLIParityKind = appparity.KindCommandSet
	HermesCLIGlobalFlag     HermesCLIParityKind = appparity.KindGlobalFlag
	HermesCLISlashCommand   HermesCLIParityKind = appparity.KindSlashCommand
	HermesCLIAlias          HermesCLIParityKind = appparity.KindAlias
	HermesCLIGatewayHandler HermesCLIParityKind = appparity.KindGatewayHandler
	HermesCLIPluginCommand  HermesCLIParityKind = appparity.KindPluginCommand
)

type HermesCLIParityEntry = appparity.Entry

func HermesCLIParityManifest() []HermesCLIParityEntry {
	return appparity.Manifest()
}
