package config

import "testing"

func TestEmbeddedRuntimeVersion(t *testing.T) {
	if got, want := GetVersion(), "v0.1.0-beta-simple"; got != want {
		t.Fatalf("GetVersion() = %q, want %q", got, want)
	}
	if got, want := GetXrayRuntimeVersion(), "26.3.27"; got != want {
		t.Fatalf("GetXrayRuntimeVersion() = %q, want %q", got, want)
	}
}
