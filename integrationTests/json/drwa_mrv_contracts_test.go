package vmjsonintegrationtest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	scenexec "github.com/multiversx/mx-chain-scenario-go/scenario/executor"
	scenio "github.com/multiversx/mx-chain-scenario-go/scenario/io"
	"github.com/multiversx/mx-chain-scenario-go/worldmock"
	contextmock "github.com/multiversx/mx-chain-vm-go/mock/context"
	vmscenario "github.com/multiversx/mx-chain-vm-go/scenario"
	"github.com/multiversx/mx-chain-vm-go/testcommon/testexecutor"
	"github.com/stretchr/testify/require"
)

func TestDRWAMRVGeneratedContractsDeployCompatibility(t *testing.T) {
	vmContractsRoot := filepath.Join(getTestRoot(), "contracts")
	require.DirExists(t, filepath.Join(vmContractsRoot, "drwa"))
	require.DirExists(t, filepath.Join(vmContractsRoot, "mrv"))

	for _, tc := range drwaMRVDeployCompatibilityCases(vmContractsRoot) {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			runDRWAMRVDeployCompatibilityScenario(t, tc)
		})
	}
}

type drwaMRVDeployCompatibilityCase struct {
	name         string
	newAddress   string
	arguments    []string
	contractKey  string
	contractMxsc string
}

func runDRWAMRVDeployCompatibilityScenario(t *testing.T, tc drwaMRVDeployCompatibilityCase) {
	t.Helper()

	scenarioPath := writeDRWAMRVDeployCompatibilityScenario(t, tc)

	vmBuilder := vmscenario.NewScenarioVMHostBuilder()
	vmBuilder.OverrideVMExecutor = testexecutor.NewDefaultTestExecutorFactory(t)

	scenarioExecutor := scenexec.NewScenarioExecutor(vmBuilder)
	defer scenarioExecutor.Close()
	scenarioExecutor.World.EnableEpochsHandler = worldmock.EnableEpochsHandlerStubAllFlags()
	authorizeDRWASyncCallers(scenarioExecutor, vmBuilder.GetVMType())

	fileResolver := scenio.NewDefaultFileResolver()
	fileResolver.ReplacePath(tc.contractKey, tc.contractMxsc)

	runner := scenio.NewScenarioController(
		scenarioExecutor,
		fileResolver,
		vmBuilder.GetVMType(),
	)
	require.NoError(t, runner.RunSingleJSONScenario(scenarioPath, scenio.DefaultRunScenarioOptions()))
}

func writeDRWAMRVDeployCompatibilityScenario(t *testing.T, tc drwaMRVDeployCompatibilityCase) string {
	t.Helper()

	dir := t.TempDir()
	scenarioPath := filepath.Join(dir, tc.name+".scen.json")

	arguments := make([]string, 0, len(tc.arguments))
	for _, arg := range tc.arguments {
		arguments = append(arguments, fmt.Sprintf("%q", arg))
	}

	scenario := fmt.Sprintf(`{
  "name": %q,
  "steps": [
    {
      "step": "setState",
      "accounts": {
        "address:owner": { "nonce": "0", "balance": "1,000,000,000,000,000,000" },
        "address:governance": { "nonce": "0", "balance": "1,000,000,000,000,000,000" },
        "address:signer1": { "nonce": "0", "balance": "1,000,000,000,000,000,000" },
        "address:signer2": { "nonce": "0", "balance": "1,000,000,000,000,000,000" },
        "address:signer3": { "nonce": "0", "balance": "1,000,000,000,000,000,000" },
        "address:signer4": { "nonce": "0", "balance": "1,000,000,000,000,000,000" },
        "address:signer5": { "nonce": "0", "balance": "1,000,000,000,000,000,000" },
        "address:buffer-pool": { "nonce": "0", "balance": "0" },
        "address:carbon-credit": { "nonce": "0", "balance": "0" }
      },
      "newAddresses": [
        {
          "creatorAddress": "address:owner",
          "creatorNonce": "0",
          "newAddress": %q
        }
      ],
      "currentBlockInfo": {
        "blockRound": "1",
        "blockTimestamp": "1000"
      }
    },
    {
      "step": "scDeploy",
      "id": "deploy",
      "tx": {
        "from": "address:owner",
        "contractCode": "mxsc:%s",
        "arguments": [%s],
        "gasLimit": "100,000,000",
        "gasPrice": "0"
      },
      "expect": {
        "out": "*",
        "status": "0",
        "gas": "*",
        "refund": "*"
      }
    }
  ]
}
`, tc.name, tc.newAddress, tc.contractKey, strings.Join(arguments, ", "))

	require.NoError(t, os.WriteFile(scenarioPath, []byte(scenario), 0o600))
	return scenarioPath
}

func authorizeDRWASyncCallers(executor *scenexec.ScenarioExecutor, vmType []byte) {
	executor.World.ProvidedBlockchainHook = &contextmock.BlockchainHookStub{
		ApplyDRWASyncEnvelopeBytesCalled: func(_ []byte, _ []byte) error {
			return nil
		},
	}
	executor.World.AuthorizedDRWASyncCallers = make(map[string]struct{})
	for _, caller := range []string{
		"drwa_policy_registry",
		"drwa_asset_manager",
		"drwa_attestation",
		"drwa_identity_registry",
		"drwa_auth_admin",
	} {
		executor.World.AuthorizedDRWASyncCallers[string(makeScenarioSCAddress(caller, vmType))] = struct{}{}
	}
}

func makeScenarioSCAddress(prefix string, vmType []byte) []byte {
	const (
		scAddressNumLeadingZeros     = 8
		scAddressReservedPrefixBytes = 10
		addressLen                   = 32
	)

	address := make([]byte, addressLen)
	copy(address[scAddressReservedPrefixBytes:], []byte(prefix))
	for i := scAddressReservedPrefixBytes + len(prefix); i < addressLen; i++ {
		address[i] = '_'
	}
	copy(address[scAddressNumLeadingZeros:scAddressReservedPrefixBytes], vmType)
	return address
}

func drwaMRVDeployCompatibilityCases(vmContractsRoot string) []drwaMRVDeployCompatibilityCase {
	pathFor := func(parts ...string) string {
		return filepath.Join(append([]string{vmContractsRoot}, parts...)...)
	}

	return []drwaMRVDeployCompatibilityCase{
		{
			name:        "drwa-policy-registry",
			newAddress:  "sc:drwa_policy_registry",
			arguments:   []string{"address:governance"},
			contractKey: "drwa-policy-registry.mxsc.json",
			contractMxsc: pathFor(
				"drwa", "policy-registry", "output", "drwa-policy-registry.mxsc.json"),
		},
		{
			name:        "drwa-asset-manager",
			newAddress:  "sc:drwa_asset_manager",
			arguments:   []string{"address:governance"},
			contractKey: "drwa-asset-manager.mxsc.json",
			contractMxsc: pathFor(
				"drwa", "asset-manager", "output", "drwa-asset-manager.mxsc.json"),
		},
		{
			name:        "drwa-attestation",
			newAddress:  "sc:drwa_attestation",
			arguments:   []string{"address:governance"},
			contractKey: "drwa-attestation.mxsc.json",
			contractMxsc: pathFor(
				"drwa", "attestation", "output", "drwa-attestation.mxsc.json"),
		},
		{
			name:        "drwa-identity-registry",
			newAddress:  "sc:drwa_identity_registry",
			arguments:   []string{"address:governance"},
			contractKey: "drwa-identity-registry.mxsc.json",
			contractMxsc: pathFor(
				"drwa", "identity-registry", "output", "drwa-identity-registry.mxsc.json"),
		},
		{
			name:        "drwa-auth-admin",
			newAddress:  "sc:drwa_auth_admin",
			arguments:   []string{"3", "100", "address:signer1", "address:signer2", "address:signer3", "address:signer4", "address:signer5"},
			contractKey: "drwa-auth-admin.mxsc.json",
			contractMxsc: pathFor(
				"drwa", "drwa-auth-admin", "output", "drwa-auth-admin.mxsc.json"),
		},
		{
			name:        "mrv-aggregator",
			newAddress:  "sc:mrv_aggregator",
			arguments:   []string{"2", "172800", "864000", "2592000", "3000"},
			contractKey: "mrv-aggregator.mxsc.json",
			contractMxsc: pathFor(
				"mrv", "aggregator", "output", "mrv-aggregator.mxsc.json"),
		},
		{
			name:        "mrv-atomic-swap",
			newAddress:  "sc:mrv_atomic_swap",
			arguments:   []string{"str:COME-abcdef"},
			contractKey: "mrv-atomic-swap.mxsc.json",
			contractMxsc: pathFor(
				"mrv", "atomic-swap", "output", "mrv-atomic-swap.mxsc.json"),
		},
		{
			name:        "mrv-buffer-pool",
			newAddress:  "sc:mrv_buffer_pool",
			arguments:   []string{"address:governance", "address:carbon-credit"},
			contractKey: "mrv-buffer-pool.mxsc.json",
			contractMxsc: pathFor(
				"mrv", "buffer-pool", "output", "mrv-buffer-pool.mxsc.json"),
		},
		{
			name:        "mrv-carbon-credit",
			newAddress:  "sc:mrv_carbon_credit",
			arguments:   []string{"address:governance", "address:buffer-pool"},
			contractKey: "mrv-carbon-credit.mxsc.json",
			contractMxsc: pathFor(
				"mrv", "carbon-credit", "output", "mrv-carbon-credit.mxsc.json"),
		},
		{
			name:        "mrv-come-settlement",
			newAddress:  "sc:mrv_come_settlement",
			arguments:   []string{"address:governance"},
			contractKey: "mrv-come-settlement.mxsc.json",
			contractMxsc: pathFor(
				"mrv", "come-settlement", "output", "mrv-come-settlement.mxsc.json"),
		},
		{
			name:        "mrv-governance-multisig",
			newAddress:  "sc:mrv_gov_multi",
			arguments:   []string{"2", "address:signer1", "address:signer2"},
			contractKey: "mrv-governance-multisig.mxsc.json",
			contractMxsc: pathFor(
				"mrv", "governance-multisig", "output", "mrv-governance-multisig.mxsc.json"),
		},
		{
			name:        "mrv-governance",
			newAddress:  "sc:mrv_governance",
			arguments:   []string{"1", "3600", "address:signer1", "address:signer2"},
			contractKey: "mrv-governance.mxsc.json",
			contractMxsc: pathFor(
				"mrv", "governance", "output", "mrv-governance.mxsc.json"),
		},
		{
			name:        "mrv-gsoc-registry",
			newAddress:  "sc:mrv_gsoc_registry",
			arguments:   []string{"address:governance"},
			contractKey: "mrv-gsoc-registry.mxsc.json",
			contractMxsc: pathFor(
				"mrv", "gsoc-registry", "output", "mrv-gsoc-registry.mxsc.json"),
		},
		{
			name:        "mrv-income-distribution",
			newAddress:  "sc:mrv_income_dist",
			arguments:   []string{"address:governance", "str:COME-abcdef"},
			contractKey: "mrv-income-distribution.mxsc.json",
			contractMxsc: pathFor(
				"mrv", "income-distribution", "output", "mrv-income-distribution.mxsc.json"),
		},
		{
			name:        "mrv-registry",
			newAddress:  "sc:mrv_registry",
			arguments:   []string{"address:governance"},
			contractKey: "mrv-registry.mxsc.json",
			contractMxsc: pathFor(
				"mrv", "registry", "output", "mrv-registry.mxsc.json"),
		},
		{
			name:        "mrv-reserve-proof-registry",
			newAddress:  "sc:mrv_reserve_proof",
			arguments:   []string{"address:governance"},
			contractKey: "mrv-reserve-proof-registry.mxsc.json",
			contractMxsc: pathFor(
				"mrv", "reserve-proof-registry", "output", "mrv-reserve-proof-registry.mxsc.json"),
		},
	}
}
