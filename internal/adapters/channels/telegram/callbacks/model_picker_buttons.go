package callbacks

import "strings"

const (
	modelPickerCBProvider  = "mp"
	modelPickerCBModel     = "mm"
	modelPickerCBGroupPage = "mg"
	modelPickerCBBack      = "mb"
	modelPickerCBCancel    = "mx"
)

func ParseModelPickerCallback(data string) (prefix, value string, ok bool) {
	data = strings.TrimSpace(data)
	idx := strings.IndexByte(data, ':')
	if idx < 0 {
		return "", "", false
	}
	prefix = data[:idx]
	switch prefix {
	case modelPickerCBProvider, modelPickerCBModel, modelPickerCBGroupPage, modelPickerCBBack, modelPickerCBCancel:
		return prefix, data[idx+1:], true
	}
	return "", "", false
}
