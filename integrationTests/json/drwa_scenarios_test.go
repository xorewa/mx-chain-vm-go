package vmjsonintegrationtest

import (
	"os"
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
			scenarioPath := tc.path
			if tc.writeScenario != nil {
				scenarioPath = tc.writeScenario(t)
			}

			builder := ScenariosTest(t).
				FilePath(scenarioPath).
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
	name          string
	path          string
	writeScenario func(t *testing.T) string
	minSyncCalls  int
}

func drwaScenarioSyncHookCoverageCases() []drwaScenarioSyncHookCoverageCase {
	scenariosRoot := filepath.Join(getOcelotRoot(), "mx-sdk-rs", "contracts", "drwa")
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
			name:         "asset-manager",
			path:         pathFor("asset-manager", "scenarios", "asset-manager-init.scen.json"),
			minSyncCalls: 3,
		},
		{
			name:         "attestation",
			path:         pathFor("attestation", "scenarios", "attestation-init.scen.json"),
			minSyncCalls: 1,
		},
		{
			name:          "drwa-auth-admin",
			writeScenario: writeDRWAAuthAdminCurrentScenario,
			minSyncCalls:  1,
		},
		{
			name:         "identity-registry",
			path:         pathFor("identity-registry", "scenarios", "identity-registry-init.scen.json"),
			minSyncCalls: 2,
		},
	}
}

func getOcelotRoot() string {
	return filepath.Clean(filepath.Join(getTestRoot(), "..", ".."))
}

func writeDRWAAuthAdminCurrentScenario(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	scenarioPath := filepath.Join(dir, "drwa-auth-admin-current.scen.json")
	scenario := `{
  "name": "drwa auth admin current 3-of-5 caller rotation",
  "steps": [
    {
      "step": "setState",
      "accounts": {
        "address:owner": { "nonce": "0", "balance": "1,000,000" },
        "address:signer1": { "nonce": "0", "balance": "1,000,000" },
        "address:signer2": { "nonce": "0", "balance": "1,000,000" },
        "address:signer3": { "nonce": "0", "balance": "1,000,000" },
        "address:signer4": { "nonce": "0", "balance": "1,000,000" },
        "address:signer5": { "nonce": "0", "balance": "1,000,000" }
      },
      "newAddresses": [
        {
          "creatorAddress": "address:owner",
          "creatorNonce": "0",
          "newAddress": "sc:drwa_auth_admin"
        }
      ],
      "currentBlockInfo": {
        "blockRound": "1"
      }
    },
    {
      "step": "scDeploy",
      "id": "deploy",
      "tx": {
        "from": "address:owner",
        "contractCode": "mxsc:../output/drwa-auth-admin.mxsc.json",
        "arguments": [
          "3",
          "20000",
          "address:signer1",
          "address:signer2",
          "address:signer3",
          "address:signer4",
          "address:signer5"
        ],
        "gasLimit": "80,000,000",
        "gasPrice": "0"
      },
      "expect": {
        "status": "0",
        "out": [],
        "gas": "*",
        "refund": "*"
      }
    },
    {
      "step": "scCall",
      "id": "propose-update",
      "tx": {
        "from": "address:signer1",
        "to": "sc:drwa_auth_admin",
        "function": "proposeUpdateCallerAddress",
        "arguments": [
          "str:auth_admin",
          "str:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
        ],
        "gasLimit": "80,000,000",
        "gasPrice": "0"
      },
      "expect": {
        "status": "0",
        "out": ["1"],
        "gas": "*",
        "refund": "*"
      }
    },
    {
      "step": "scCall",
      "id": "sign-update-2",
      "tx": {
        "from": "address:signer2",
        "to": "sc:drwa_auth_admin",
        "function": "sign",
        "arguments": ["1"],
        "gasLimit": "50,000,000",
        "gasPrice": "0"
      },
      "expect": {
        "status": "0",
        "out": [],
        "gas": "*",
        "refund": "*"
      }
    },
    {
      "step": "scCall",
      "id": "sign-update-3",
      "tx": {
        "from": "address:signer3",
        "to": "sc:drwa_auth_admin",
        "function": "sign",
        "arguments": ["1"],
        "gasLimit": "50,000,000",
        "gasPrice": "0"
      },
      "expect": {
        "status": "0",
        "out": [],
        "gas": "*",
        "refund": "*"
      }
    },
    {
      "step": "setState",
      "currentBlockInfo": {
        "blockRound": "15000"
      }
    },
    {
      "step": "scCall",
      "id": "perform-update",
      "tx": {
        "from": "address:signer1",
        "to": "sc:drwa_auth_admin",
        "function": "performAction",
        "arguments": ["1"],
        "gasLimit": "100,000,000",
        "gasPrice": "0"
      },
      "expect": {
        "status": "0",
        "out": "*",
        "gas": "*",
        "refund": "*"
      }
    },
    {
      "step": "scQuery",
      "id": "verify-version",
      "tx": {
        "to": "sc:drwa_auth_admin",
        "function": "getAuthorizedCallerVersion",
        "arguments": ["str:auth_admin"]
      },
      "expect": {
        "status": "0",
        "out": ["1"]
      }
    }
  ]
}
`

	if err := os.WriteFile(scenarioPath, []byte(scenario), 0o600); err != nil {
		t.Fatal(err)
	}
	return scenarioPath
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
