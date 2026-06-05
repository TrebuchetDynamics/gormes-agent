package tasks

import "testing"

func TestCuratorAuxiliarySlot_ModelPickerTaskRegistry(t *testing.T) {
	tasks := DefaultAuxiliaryTaskEntries()
	for _, task := range tasks {
		if task.Key == "curator" {
			if task.Label == "" || task.Description == "" {
				t.Fatalf("curator task = %#v, want label and description", task)
			}
			return
		}
	}
	t.Fatalf("DefaultAuxiliaryTaskEntries missing curator: %#v", tasks)
}
