package store

import "testing"

func TestStoreExistsForPackageCoverage(t *testing.T) {
	if New(nil).DB() != nil {
		t.Fatal("nil db should remain nil")
	}
}
