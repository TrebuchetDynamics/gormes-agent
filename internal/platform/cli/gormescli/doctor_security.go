package gormescli

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/doctor"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/security"
)

// DoctorSecurityAdvisoriesStatus binds the importable doctor advisory renderer
// to the Gormes-owned security catalog and ack store.
func DoctorSecurityAdvisoriesStatus(ackID, home string) doctor.CheckResult {
	store := security.NewAckStore(home)

	var ackConfirm string
	if id := strings.TrimSpace(ackID); id != "" {
		if err := store.Ack(id); err != nil {
			ackConfirm = fmt.Sprintf("could not record ack for %q: %v", id, err)
		} else {
			ackConfirm = fmt.Sprintf("acknowledged %s (recorded under ~/.gormes)", id)
		}
	}

	acked, _ := store.AckedIDs()
	hits := security.DetectCompromised(security.DefaultCatalog(), security.NoInstalledPackages)
	views := make([]doctor.DoctorAdvisoryView, 0, len(hits))
	for _, h := range hits {
		_, isAcked := acked[h.Advisory.ID]
		views = append(views, doctor.DoctorAdvisoryView{
			ID:          h.Advisory.ID,
			Title:       h.Advisory.Title,
			Package:     h.Package,
			Version:     h.InstalledVersion,
			Remediation: security.FullRemediationText(h),
			Acked:       isAcked,
		})
	}

	res := doctor.CheckSecurityAdvisories(doctor.DoctorSecurityAdvisoryInventory{Hits: views})
	if ackConfirm != "" {
		res.Items = append(res.Items, doctor.ItemInfo{
			Name:   "ack",
			Status: doctor.StatusPass,
			Note:   ackConfirm,
		})
	}
	return res
}
