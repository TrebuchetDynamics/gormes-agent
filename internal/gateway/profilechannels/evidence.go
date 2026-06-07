package profilechannels

func HasEvidenceCode(items []ProfileChannelReadinessEvidence, code string) bool {
	for _, item := range items {
		if item.Code == code {
			return true
		}
	}
	return false
}
