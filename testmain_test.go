package spanalyzer

import (
	"os"
	"runtime/debug"
	"testing"
)

func TestMain(m *testing.M) {
	// go-googlesql owns GoogleSQL frontend objects in WASM and releases many
	// wrappers through Go finalizers. The root package test suite constructs
	// many short-lived catalogs, including custom Spanner functions and system
	// tables. Those finalizers can run while borrowed WASM-side catalog objects
	// are still reachable from other wrappers, which shows up as delayed
	// out-of-bounds accesses in unrelated tests. Keep GC disabled for this test
	// process so the suite exercises analyzer behavior without depending on
	// finalizer timing.
	oldGCPercent := debug.SetGCPercent(-1)
	code := m.Run()
	debug.SetGCPercent(oldGCPercent)
	os.Exit(code)
}
