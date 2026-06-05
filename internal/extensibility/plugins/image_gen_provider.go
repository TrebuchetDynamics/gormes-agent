package plugins

// ImageGenProviderCapabilities returns the plugin capability rows that declare
// image-generation providers. It is metadata-only: callers can expose provider
// availability without importing or executing plugin runtime code.
func ImageGenProviderCapabilities(inventory Inventory) []CapabilityStatus {
	out := make([]CapabilityStatus, 0, len(inventory.Capabilities))
	for _, row := range inventory.Capabilities {
		if row.Kind != CapabilityImageGen {
			continue
		}
		out = append(out, cloneCapabilityStatus(row))
	}
	sortCapabilityStatuses(out)
	return out
}

func cloneCapabilityStatus(in CapabilityStatus) CapabilityStatus {
	out := in
	out.Evidence = append([]Evidence(nil), in.Evidence...)
	return out
}
