package store

import (
	os "os"
	testing "testing"
)

// Test is a trivial test to check file path generation (dummy example)
func TestDummy(t *testing.T) {
	// dummy test to avoid import error
	_ = os.TempDir()
}
