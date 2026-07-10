package vmhooksgenerate

import (
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

func TestWriteRustCapiVMHooksEmitsUpstreamMemoryBridgeConversions(t *testing.T) {
	t.Parallel()

	outputDir := t.TempDir()
	outputFile := "capi_vm_hooks.rs"
	out := NewEIGenWriter(outputDir, outputFile)
	WriteRustCapiVMHooks(out, &EIMetadata{})
	out.Close()

	generated, err := os.ReadFile(filepath.Join(outputDir, outputFile))
	if err != nil {
		t.Fatal(err)
	}

	generatedCode := string(generated)
	requiredSnippets := []string{
		"mem_ptr as i32",
		"mem_length as i32",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(generatedCode, snippet) {
			t.Fatalf("generated C-API VM hooks missing upstream memory bridge conversion %q", snippet)
		}
	}

	forbiddenSnippets := []string{
		"capi_mem_conversion",
		"mem_ptr_to_i32(mem_ptr)",
		"mem_length_to_i32(mem_length)",
	}
	for _, snippet := range forbiddenSnippets {
		if strings.Contains(generatedCode, snippet) {
			t.Fatalf("generated C-API VM hooks contain non-upstream memory bridge conversion %q", snippet)
		}
	}
}

func TestWriteRustCapiVMHooksDoesNotEmitThreadSafetyAssertions(t *testing.T) {
	t.Parallel()

	outputDir := t.TempDir()
	outputFile := "capi_vm_hooks.rs"
	out := NewEIGenWriter(outputDir, outputFile)
	WriteRustCapiVMHooks(out, &EIMetadata{})
	out.Close()

	generated, err := os.ReadFile(filepath.Join(outputDir, outputFile))
	if err != nil {
		t.Fatal(err)
	}

	generatedCode := string(generated)
	forbiddenSnippets := []string{
		"unsafe impl Send for CapiVMHooks",
		"unsafe impl Sync for CapiVMHooks",
	}
	for _, snippet := range forbiddenSnippets {
		if strings.Contains(generatedCode, snippet) {
			t.Fatalf("generated C-API VM hooks contain hand-reviewed thread-safety assertion %q", snippet)
		}
	}
}

func TestGeneratedHookListsMatchCheckedInGoBridge(t *testing.T) {
	t.Parallel()

	eiMetadata := readCanonicalEIMetadata(t)
	wasmer2ImportsCgo := readFile(t, filepath.Join("..", "..", "..", "wasmer2", "wasmer2ImportsCgo.go"))

	expectedPointerFields, expectedCgoExports := expectedGoBridgeHookNames(eiMetadata)

	assertStringSetEqual(
		t,
		"wasmer2 cgo pointer fields",
		expectedPointerFields,
		extractMatches(wasmer2ImportsCgo, regexp.MustCompile(`\b([a-z0-9_]+_func_ptr):\s+funcPointer\(C\.w2_`)),
	)
	assertStringSetEqual(
		t,
		"wasmer2 cgo exported hook functions",
		expectedCgoExports,
		extractMatches(wasmer2ImportsCgo, regexp.MustCompile(`(?m)^//export\s+(w2_[A-Za-z0-9_]+)$`)),
	)
}

func TestRustHookListsOptionalRepoCopyIsInSync(t *testing.T) {
	t.Parallel()

	rustRepoPath := os.Getenv("MX_VM_EXECUTOR_RS_PATH")
	if rustRepoPath == "" {
		t.Skip("set MX_VM_EXECUTOR_RS_PATH to a local mx-vm-executor-rs checkout to enable Rust hook-list drift check")
	}

	eiMetadata := readCanonicalEIMetadata(t)
	expectedPointerFields, expectedImportNames := expectedRustBridgeHookNames(eiMetadata)

	rustCapiPointers := readFile(t, filepath.Join(rustRepoPath, "c-api", "src", "capi_vm_hook_pointers.rs"))
	assertStringSetEqualWithAllowedExtra(
		t,
		"Rust C-API hook pointer fields",
		expectedPointerFields,
		extractMatches(rustCapiPointers, regexp.MustCompile(`(?m)^\s+pub\s+([a-z0-9_]+_func_ptr):\s+extern\s+"C"\s+fn`)),
		parkedNativeHookPointerFields(),
	)

	rustWasmerImports := readFile(t, filepath.Join(rustRepoPath, "vm-executor-wasmer", "src", "wasmer_imports.rs"))
	assertStringSetEqualWithAllowedExtra(
		t,
		"Rust Wasmer import names",
		expectedImportNames,
		extractMatches(rustWasmerImports, regexp.MustCompile(`(?m)^\s+"([^"]+)"\s+=>\s+Function::new_native_with_env`)),
		parkedNativeHookImportNames(),
	)
}

func readCanonicalEIMetadata(t *testing.T) *EIMetadata {
	t.Helper()

	eiMetadata := &EIMetadata{
		Groups: []*EIGroup{
			{SourcePath: "baseOps.go", Name: "Main"},
			{SourcePath: "managedei.go", Name: "Managed"},
			{SourcePath: "bigFloatOps.go", Name: "BigFloat"},
			{SourcePath: "bigIntOps.go", Name: "BigInt"},
			{SourcePath: "manBufOps.go", Name: "ManagedBuffer"},
			{SourcePath: "manMapOps.go", Name: "ManagedMap"},
			{SourcePath: "smallIntOps.go", Name: "SmallInt"},
			{SourcePath: "cryptoei.go", Name: "Crypto"},
			{SourcePath: "unsafeOps.go", Name: "Unsafe"},
		},
	}

	if err := ReadAndParseEIMetadata(token.NewFileSet(), "../", eiMetadata); err != nil {
		t.Fatal(err)
	}

	return eiMetadata
}

func expectedGoBridgeHookNames(eiMetadata *EIMetadata) ([]string, []string) {
	pointerFields := make([]string, 0, len(eiMetadata.AllFunctions))
	cgoExports := make([]string, 0, len(eiMetadata.AllFunctions))
	writer := &cgoWriter{cgoPrefix: "w2_"}

	for _, funcMetadata := range eiMetadata.AllFunctions {
		pointerFields = append(pointerFields, cgoFuncPointerFieldName(funcMetadata))
		cgoExports = append(cgoExports, writer.cgoFuncName(funcMetadata))
	}

	return sortedUnique(pointerFields), sortedUnique(cgoExports)
}

func expectedRustBridgeHookNames(eiMetadata *EIMetadata) ([]string, []string) {
	pointerFields := make([]string, 0, len(eiMetadata.AllFunctions))
	importNames := make([]string, 0, len(eiMetadata.AllFunctions))

	for _, funcMetadata := range eiMetadata.AllFunctions {
		pointerFields = append(pointerFields, cgoFuncPointerFieldName(funcMetadata))
		importNames = append(importNames, lowerInitial(funcMetadata.Name))
	}

	return sortedUnique(pointerFields), sortedUnique(importNames)
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return string(content)
}

func extractMatches(content string, expression *regexp.Regexp) []string {
	matches := expression.FindAllStringSubmatch(content, -1)
	result := make([]string, 0, len(matches))
	for _, match := range matches {
		result = append(result, match[1])
	}

	return sortedUnique(result)
}

func sortedUnique(values []string) []string {
	sort.Strings(values)
	result := values[:0]
	var previous string
	for index, value := range values {
		if index == 0 || value != previous {
			result = append(result, value)
		}
		previous = value
	}

	return result
}

func assertStringSetEqual(t *testing.T, label string, expected []string, actual []string) {
	t.Helper()

	assertStringSetEqualWithAllowedExtra(t, label, expected, actual, nil)
}

func assertStringSetEqualWithAllowedExtra(t *testing.T, label string, expected []string, actual []string, allowedExtra []string) {
	t.Helper()

	missing := setDifference(expected, actual)
	extra := setDifference(setDifference(actual, expected), allowedExtra)
	if len(missing) == 0 && len(extra) == 0 {
		return
	}

	t.Fatalf("%s drift detected\nmissing: %s\nextra: %s", label, strings.Join(missing, ", "), strings.Join(extra, ", "))
}

func setDifference(left []string, right []string) []string {
	rightSet := make(map[string]struct{}, len(right))
	for _, value := range right {
		rightSet[value] = struct{}{}
	}

	var difference []string
	for _, value := range left {
		if _, ok := rightSet[value]; !ok {
			difference = append(difference, value)
		}
	}

	return difference
}

func parkedNativeHookPointerFields() []string {
	// These Rust-only native hook placeholders are tracked under parked ZK/native
	// hook work. Keep the allowlist explicit so any new drift fails review.
	return []string{
		"managed_add_ec_func_ptr",
		"managed_map_to_curve_ec_func_ptr",
		"managed_mul_ec_func_ptr",
		"managed_multi_exp_ec_func_ptr",
		"managed_pairing_checks_ec_func_ptr",
		"managed_verify_groth16_func_ptr",
		"managed_verify_plonk_func_ptr",
	}
}

func parkedNativeHookImportNames() []string {
	// Must stay in sync with parkedNativeHookPointerFields.
	return []string{
		"managedAddEC",
		"managedMapToCurveEC",
		"managedMulEC",
		"managedMultiExpEC",
		"managedPairingChecksEC",
		"managedVerifyGroth16",
		"managedVerifyPlonk",
	}
}
