package optparam

import (
	"os"
	"runtime/debug"
	"testing"
)

func TestMain(m *testing.M) {
	// Mirror internal/querygen/testmain_test.go: many short-lived GoogleSQL
	// frontend objects, so disable GC for this process to avoid WASM
	// finalizer timing issues.
	old := debug.SetGCPercent(-1)
	code := m.Run()
	debug.SetGCPercent(old)
	os.Exit(code)
}
