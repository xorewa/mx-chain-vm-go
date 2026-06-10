package wasmer2

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

// TestCWasmerAPIVersion_MatchesExpected — ISSUE-020. The single
// authoritative liveness check for the FFI ABI handshake. If the linked
// .so / .dylib reports a different version than the Go bridge expects,
// this fails immediately and the build refresh story (header + lib +
// expectedAPIVersion bump) is the obvious diff.
func TestCWasmerAPIVersion_MatchesExpected(t *testing.T) {
	t.Parallel()

	got := cWasmerAPIVersion()
	if got != expectedAPIVersion {
		t.Fatalf(
			"libvmexeccapi reports ABI v%d, Go bridge expected v%d. "+
				"Either the linked .so/.dylib is stale (refresh from "+
				"mx-vm-executor-rs/target/release/) or the bridge constant "+
				"`expectedAPIVersion` needs bumping to match the lib.",
			got, expectedAPIVersion,
		)
	}
}

// TestCheckAPIVersion_NilOnMatch — when the lib version matches, the
// guarded handshake returns nil and is safe to call repeatedly.
func TestCheckAPIVersion_NilOnMatch(t *testing.T) {
	t.Parallel()

	if err := checkAPIVersion(); err != nil {
		t.Fatalf("checkAPIVersion returned %v with matching lib; expected nil", err)
	}
	// Idempotency: a second call returns the same result without
	// re-invoking the underlying cgo call (sync.Once-guarded).
	if err := checkAPIVersion(); err != nil {
		t.Fatalf("second checkAPIVersion call returned %v; expected nil", err)
	}
}

func TestCheckAPIVersionValue_ErrorsOnMismatch(t *testing.T) {
	t.Parallel()

	var mismatched uint32
	if expectedAPIVersion == 0 {
		mismatched = 1
	} else {
		mismatched = expectedAPIVersion - 1
	}

	err := checkAPIVersionValue(mismatched)
	if err == nil {
		t.Fatalf("expected ABI mismatch error for version %d", mismatched)
	}
}

func TestLibVMExecCAPIHeader_DefineMatchesExpectedVersion(t *testing.T) {
	t.Parallel()

	headerBytes, err := os.ReadFile("libvmexeccapi.h")
	if err != nil {
		t.Fatalf("read local libvmexeccapi.h: %v", err)
	}

	version := extractHeaderAPIVersion(t, headerBytes)
	if version != expectedAPIVersion {
		t.Fatalf(
			"local libvmexeccapi.h declares ABI v%d, Go bridge expected v%d; "+
				"refresh header/lib or bump expectedAPIVersion together",
			version,
			expectedAPIVersion,
		)
	}
}

func TestLibVMExecCAPIHeader_OptionalRustRepoCopyIsInSync(t *testing.T) {
	t.Parallel()

	rustExecutorPath := os.Getenv("MX_VM_EXECUTOR_RS_PATH")
	if rustExecutorPath == "" {
		t.Skip("MX_VM_EXECUTOR_RS_PATH not set; skipping cross-repo header byte-compare")
	}

	localHeader, err := os.ReadFile("libvmexeccapi.h")
	if err != nil {
		t.Fatalf("read local libvmexeccapi.h: %v", err)
	}

	rustHeaderPath := filepath.Join(rustExecutorPath, "c-api", "libvmexeccapi.h")
	rustHeader, err := os.ReadFile(rustHeaderPath)
	if err != nil {
		t.Fatalf("read Rust executor header %q: %v", rustHeaderPath, err)
	}

	if !bytes.Equal(localHeader, rustHeader) {
		t.Fatalf("wasmer2/libvmexeccapi.h differs from %s; refresh the Go bridge header from mx-vm-executor-rs", rustHeaderPath)
	}
}

func extractHeaderAPIVersion(t *testing.T, headerBytes []byte) uint32 {
	t.Helper()

	matches := regexp.MustCompile(`(?m)^#define\s+VM_EXEC_API_VERSION\s+([0-9]+)\s*$`).FindSubmatch(headerBytes)
	if matches == nil {
		t.Fatalf("VM_EXEC_API_VERSION define not found in libvmexeccapi.h")
	}

	version, err := strconv.ParseUint(string(matches[1]), 10, 32)
	if err != nil {
		t.Fatalf("parse VM_EXEC_API_VERSION %q: %v", matches[1], err)
	}

	return uint32(version)
}
