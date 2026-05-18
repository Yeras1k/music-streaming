package testutil

import (
    "testing"
)

// TestHelper provides common test utilities
type TestHelper struct {
    t *testing.T
}

func NewTestHelper(t *testing.T) *TestHelper {
    return &TestHelper{t: t}
}
