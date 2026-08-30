package command

import "testing"

func TestEveryCatalogCommandIsAvailable(t *testing.T) {
	catalog := Catalog()
	if len(catalog) != 53 {
		t.Fatalf("catalog contains %d commands, want 53", len(catalog))
	}
	for _, spec := range catalog {
		if spec.State != Available {
			t.Errorf("%s %s is %s", spec.Category, spec.Name, spec.State)
		}
	}
}
