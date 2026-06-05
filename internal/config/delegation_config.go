package config

import delegationconfig "github.com/TrebuchetDynamics/gormes-agent/internal/config/delegation"

type DelegationCfg delegationconfig.DelegationCfg

func (d *DelegationCfg) UnmarshalTOML(data []byte) error {
	var parsed delegationconfig.DelegationCfg
	if err := parsed.UnmarshalTOML(data); err != nil {
		return err
	}
	*d = DelegationCfg(parsed)
	return nil
}
