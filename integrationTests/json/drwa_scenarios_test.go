package vmjsonintegrationtest

import (
	"path/filepath"
	"testing"
)

func TestDRWAScenarioSyncHookCoverage(t *testing.T) {
	if testing.Short() {
		t.Skip("not a short test")
	}

	for _, tc := range drwaScenarioSyncHookCoverageCases() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			builder := ScenariosTest(t).
				FilePath(tc.path).
				WithDRWABlockchainHook()
			withDRWAContractPathReplacements(builder)
			builder.Run().
				CheckNoError().
				CheckDRWASyncHookCallsAtLeast(tc.minSyncCalls)
		})
	}
}

func TestDRWAScenarioSyncHookCoverageRequiresOptInHook(t *testing.T) {
	if testing.Short() {
		t.Skip("not a short test")
	}

	scenarioPath := drwaScenarioSyncHookCoverageCases()[0].path
	builder := ScenariosTest(t).FilePath(scenarioPath)
	withDRWAContractPathReplacements(builder)
	builder.Run().
		RequireErrorContains("opcode is not allowed")
}

type drwaScenarioSyncHookCoverageCase struct {
	name         string
	path         string
	minSyncCalls int
}

func drwaScenarioSyncHookCoverageCases() []drwaScenarioSyncHookCoverageCase {
	scenariosRoot := filepath.Join(getTestRoot(), "contracts", "drwa")
	pathFor := func(parts ...string) string {
		return filepath.Join(append([]string{scenariosRoot}, parts...)...)
	}

	return []drwaScenarioSyncHookCoverageCase{
		{
			name:         "policy-registry",
			path:         pathFor("policy-registry", "scenarios", "policy-registry-init.scen.json"),
			minSyncCalls: 2,
		},
		{
			name:         "policy-registry-denial-signals",
			path:         pathFor("policy-registry", "scenarios", "policy-registry-denial-signals.scen.json"),
			minSyncCalls: 1,
		},
		{
			name:         "asset-manager",
			path:         pathFor("asset-manager", "scenarios", "asset-manager-init.scen.json"),
			minSyncCalls: 3,
		},
		{
			name:         "asset-manager-denial-signals",
			path:         pathFor("asset-manager", "scenarios", "asset-manager-denial-signals.scen.json"),
			minSyncCalls: 1,
		},
		{
			name:         "attestation",
			path:         pathFor("attestation", "scenarios", "attestation-init.scen.json"),
			minSyncCalls: 1,
		},
		{
			name:         "attestation-denial-signals",
			path:         pathFor("attestation", "scenarios", "attestation-denial-signals.scen.json"),
			minSyncCalls: 1,
		},
		{
			name:         "drwa-auth-admin",
			path:         pathFor("drwa-auth-admin", "scenarios", "drwa-auth-admin-current.scen.json"),
			minSyncCalls: 1,
		},
		{
			name:         "identity-registry",
			path:         pathFor("identity-registry", "scenarios", "identity-registry-init.scen.json"),
			minSyncCalls: 2,
		},
		{
			name:         "identity-registry-denial-signals",
			path:         pathFor("identity-registry", "scenarios", "identity-registry-denial-signals.scen.json"),
			minSyncCalls: 1,
		},
	}
}

func withDRWAContractPathReplacements(builder *ScenariosTestBuilder) {
	contractsRoot := filepath.Join(getTestRoot(), "contracts", "drwa")
	builder.
		ReplacePath("../output/drwa-policy-registry.mxsc.json",
			filepath.Join(contractsRoot, "policy-registry", "output", "drwa-policy-registry.mxsc.json")).
		ReplacePath("../../policy-registry/output/drwa-policy-registry.mxsc.json",
			filepath.Join(contractsRoot, "policy-registry", "output", "drwa-policy-registry.mxsc.json")).
		ReplacePath("../output/drwa-asset-manager.mxsc.json",
			filepath.Join(contractsRoot, "asset-manager", "output", "drwa-asset-manager.mxsc.json")).
		ReplacePath("../output/drwa-attestation.mxsc.json",
			filepath.Join(contractsRoot, "attestation", "output", "drwa-attestation.mxsc.json")).
		ReplacePath("../output/drwa-auth-admin.mxsc.json",
			filepath.Join(contractsRoot, "drwa-auth-admin", "output", "drwa-auth-admin.mxsc.json")).
		ReplacePath("../output/drwa-identity-registry.mxsc.json",
			filepath.Join(contractsRoot, "identity-registry", "output", "drwa-identity-registry.mxsc.json"))
}
