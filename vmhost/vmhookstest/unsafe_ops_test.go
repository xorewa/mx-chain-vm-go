package vmhookstest

import (
	"testing"

	"github.com/multiversx/mx-chain-scenario-go/worldmock"
	contextmock "github.com/multiversx/mx-chain-vm-go/mock/context"
	test "github.com/multiversx/mx-chain-vm-go/testcommon"
	"github.com/multiversx/mx-chain-vm-go/vmhost"
	"github.com/multiversx/mx-chain-vm-go/vmhost/vmhooks"
	"github.com/stretchr/testify/require"
)

func TestUnsafeModeRecordsConditionalErrors(t *testing.T) {
	t.Parallel()

	_, err := test.BuildMockInstanceCallTest(t).
		WithContracts(
			test.CreateMockContract(test.ParentAddress).
				WithBalance(1000).
				WithMethods(func(instanceMock *contextmock.InstanceMock, config interface{}) {
					instanceMock.AddMockMethod("testFunction", func() *contextmock.InstanceMock {
						host := instanceMock.Host
						managedTypes := host.ManagedTypes()
						hooks := vmhooks.NewVMHooksImpl(host)

						require.False(t, host.Runtime().IsUnsafeMode())
						hooks.ActivateUnsafeMode()
						require.True(t, host.Runtime().IsUnsafeMode())
						require.Equal(t, int32(0), hooks.ManagedGetNumErrors())

						errorHandle := managedTypes.NewManagedBuffer()
						hooks.ManagedGetErrorWithIndex(0, errorHandle)
						require.Equal(t, int32(1), hooks.ManagedGetNumErrors())

						hooks.ManagedGetErrorWithIndex(0, errorHandle)
						errorMessage, err := managedTypes.GetBytes(errorHandle)
						require.NoError(t, err)
						require.Equal(t, vmhost.ErrInvalidArgument.Error(), string(errorMessage))

						lastErrorHandle := managedTypes.NewManagedBuffer()
						hooks.ManagedGetLastError(lastErrorHandle)
						lastErrorMessage, err := managedTypes.GetBytes(lastErrorHandle)
						require.NoError(t, err)
						require.Equal(t, vmhost.ErrInvalidArgument.Error(), string(lastErrorMessage))

						hooks.DeactivateUnsafeMode()
						require.False(t, host.Runtime().IsUnsafeMode())

						return instanceMock
					})
				}),
		).
		WithInput(test.CreateTestContractCallInputBuilder().
			WithRecipientAddr(test.ParentAddress).
			WithGasProvided(100000).
			WithFunction("testFunction").
			Build()).
		AndAssertResults(func(world *worldmock.MockWorld, verify *test.VMOutputVerifier) {
			verify.Ok()
		})
	require.NoError(t, err)
}
