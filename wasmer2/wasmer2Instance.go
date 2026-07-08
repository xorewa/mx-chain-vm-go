package wasmer2

// #include <stdlib.h>
import "C"
import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"unsafe"

	"github.com/multiversx/mx-chain-vm-go/executor"
)

var _ executor.Instance = (*Wasmer2Instance)(nil)

// Wasmer2Instance represents a WebAssembly instance.
type Wasmer2Instance struct {
	// The underlying WebAssembly instance.
	cgoInstance *cWasmerInstanceT

	// The exported memory of a WebAssembly instance.
	memory Wasmer2Memory

	AlreadyClean bool

	// ISSUE-082: cleanMu serializes Clean() / IsAlreadyCleaned() / Reset()
	// reads of AlreadyClean. Without the lock, two concurrent Clean() calls
	// could both observe AlreadyClean=false and both call cWasmerInstanceDestroy.
	// The Rust handle_registry absorbs the race today (HashMap::remove is
	// idempotent), but relying on that is fragile if the Rust API contract
	// ever tightens. Mutex was chosen over sync.Once to preserve the public
	// AlreadyClean field for backward compat with tests at
	// vmhost/contexts/instanceTracker_test.go:17,341 and
	// mock/context/instanceMock.go that initialize the field via struct
	// literal. Zero-value sync.Mutex is valid, so existing struct-literal
	// initializers remain compatible.
	cleanMu sync.Mutex
}

func emptyInstance() *Wasmer2Instance {
	return &Wasmer2Instance{cgoInstance: nil}
}

func newInstance(c_instance *cWasmerInstanceT) (*Wasmer2Instance, error) {
	return &Wasmer2Instance{
		cgoInstance: c_instance,
		memory: Wasmer2Memory{
			cgoInstance: c_instance,
		},
	}, nil
}

// Clean cleans instance.
//
// ISSUE-082: serialized via instance.cleanMu so two concurrent Clean() calls
// cannot both observe AlreadyClean=false and both call cWasmerInstanceDestroy.
// The Rust handle_registry is idempotent on double-destroy today, but
// relying on that is fragile if the contract ever tightens.
func (instance *Wasmer2Instance) Clean() bool {
	instance.cleanMu.Lock()
	defer instance.cleanMu.Unlock()

	logWasmer2.Trace("cleaning instance", "id", instance.ID())
	if instance.AlreadyClean {
		logWasmer2.Trace("clean: already cleaned instance", "id", instance.ID())
		return false
	}

	if instance.cgoInstance != nil {
		cWasmerInstanceDestroy(instance.cgoInstance)

		instance.AlreadyClean = true
		logWasmer2.Trace("cleaned instance", "id", instance.ID())

		return true
	}

	return false
}

// IsAlreadyCleaned returns the internal field AlreadyClean.
//
// ISSUE-082: takes instance.cleanMu so concurrent reads are race-free under
// the Go race detector. The lock is released before return, so the value
// is a snapshot that can become stale immediately if another goroutine
// calls Clean() after this returns — but that is the inherent semantic
// of any "is X" predicate on shared mutable state.
func (instance *Wasmer2Instance) IsAlreadyCleaned() bool {
	instance.cleanMu.Lock()
	defer instance.cleanMu.Unlock()
	return instance.AlreadyClean
}

// SetGasLimit sets the gas limit for the instance
func (instance *Wasmer2Instance) SetGasLimit(gasLimit uint64) {
	cWasmerInstanceSetGasLimit(instance.cgoInstance, gasLimit)
}

// SetPointsUsed sets the internal instance gas counter
func (instance *Wasmer2Instance) SetPointsUsed(points uint64) {
	cWasmerInstanceSetPointsUsed(instance.cgoInstance, points)
}

// GetPointsUsed returns the internal instance gas counter
func (instance *Wasmer2Instance) GetPointsUsed() uint64 {
	return cWasmerInstanceGetPointsUsed(instance.cgoInstance)
}

// SetBreakpointValue sets the breakpoint value for the instance
func (instance *Wasmer2Instance) SetBreakpointValue(value uint64) {
	cWasmerInstanceSetBreakpointValue(instance.cgoInstance, value)
}

// GetBreakpointValue returns the breakpoint value
func (instance *Wasmer2Instance) GetBreakpointValue() uint64 {
	return cWasmerInstanceGetBreakpointValue(instance.cgoInstance)
}

// Cache caches the instance
func (instance *Wasmer2Instance) Cache() ([]byte, error) {
	var cacheBytes *cUchar
	var cacheLen cUint32T

	var cacheResult = cWasmerInstanceCache(
		instance.cgoInstance,
		&cacheBytes,
		&cacheLen,
	)

	if cacheResult != cWasmerOk {
		return nil, ErrCachingFailed
	}

	goBytes := C.GoBytes(unsafe.Pointer(cacheBytes), C.int(cacheLen))
	cFree(unsafe.Pointer(cacheBytes))
	cacheBytes = nil
	return goBytes, nil
}

// IsFunctionImported returns true if the instance imports the specified function
func (instance *Wasmer2Instance) IsFunctionImported(name string) bool {
	var cImportName = cCString(name)
	defer cFree(unsafe.Pointer(cImportName))

	result := cWasmerInstanceHasImportedFunction(
		instance.cgoInstance,
		cImportName,
	)

	return result == 1
}

// CallFunction executes given function from loaded contract.
func (instance *Wasmer2Instance) CallFunction(functionName string) error {
	var wasmFunctionName = cCString(functionName)
	defer cFree(unsafe.Pointer(wasmFunctionName))

	var callResult = cWasmerInstanceCall(
		instance.cgoInstance,
		wasmFunctionName,
	)

	if callResult != cWasmerOk {
		err := fmt.Errorf("failed to call the `%s` exported function", functionName)
		return newWrappedError(err)
	}

	return nil
}

// HasFunction checks if loaded contract has a function (endpoint) with given name.
func (instance *Wasmer2Instance) HasFunction(functionName string) bool {
	var wasmFunctionName = cCString(functionName)
	defer cFree(unsafe.Pointer(wasmFunctionName))

	result := cWasmerInstanceHasFunction(
		instance.cgoInstance,
		wasmFunctionName,
	)

	return result == 1
}

// GetLastError returns the last error message if any, otherwise returns an error.
func (instance *Wasmer2Instance) getFunctionNamesConcat() (string, error) {
	var bufferLength = cWasmerInstanceExportedFunctionNamesLength(instance.cgoInstance)

	// ISSUE-008: the Rust side now signals overflow / null-instance with
	// -1 (it was previously conflated with the 0-length "no functions"
	// case). Detect the negative sentinel BEFORE allocating a buffer —
	// make([]T, negative) would panic at runtime.
	if bufferLength < 0 {
		return "", errors.New("cannot read function names: length signal indicates error")
	}

	if bufferLength == 0 {
		return "", nil
	}

	var buffer = make([]cChar, bufferLength)
	var bufferPointer = (*cChar)(unsafe.Pointer(&buffer[0]))

	var result = cWasmerInstanceExportedFunctionNames(instance.cgoInstance, bufferPointer, bufferLength)

	if result == -1 {
		return "", errors.New("cannot read function names")
	}

	return cGoString(bufferPointer), nil
}

// GetFunctionNames returns a list of the function names exported by the contract.
func (instance *Wasmer2Instance) GetFunctionNames() []string {
	buffer, err := instance.getFunctionNamesConcat()
	if err != nil {
		return nil
	}
	return strings.Split(buffer, "|")
}

// ValidateFunctionArities checks that no function (endpoint) of the given contract has any parameters or returns any result.
// All arguments and results should be transferred via the import functions.
func (instance *Wasmer2Instance) ValidateFunctionArities() error {
	var result = cWasmerCheckSignatures(instance.cgoInstance)
	if result != cWasmerOk {
		return executor.ErrFunctionNonvoidSignature
	}
	return nil
}

// HasMemory checks whether the instance has at least one exported memory.
func (instance *Wasmer2Instance) HasMemory() bool {
	return true
}

// MemLoad returns the contents from the given offset of the WASM memory.
func (instance *Wasmer2Instance) MemLoad(memPtr executor.MemPtr, length executor.MemLength) ([]byte, error) {
	return executor.MemLoadFromMemory(&instance.memory, memPtr, length)
}

// MemStore stores the given data in the WASM memory at the given offset.
func (instance *Wasmer2Instance) MemStore(memPtr executor.MemPtr, data []byte) error {
	return executor.MemStoreToMemory(&instance.memory, memPtr, data)
}

// MemLength returns the length of the allocated memory. Only called directly in tests.
func (instance *Wasmer2Instance) MemLength() uint32 {
	return instance.memory.Length()
}

// MemGrow allocates more pages to the current memory. Only called directly in tests.
func (instance *Wasmer2Instance) MemGrow(pages uint32) error {
	return instance.memory.Grow(pages)
}

// MemDump yields the entire contents of the memory. Only used in tests.
//
// ISSUE-003: migrated from Wasmer2Memory.Data() (which returns an alias
// over wasmer linear memory and dangles after memory.Grow) to
// ReadMemory which returns a defensive copy with bounds checking.
// MemDump callers always store the result for later inspection, so the
// stable-copy semantics matter.
func (instance *Wasmer2Instance) MemDump() []byte {
	length := instance.memory.Length()
	if length == 0 {
		return []byte{}
	}
	dump, err := instance.memory.ReadMemory(0, length)
	if err != nil {
		// MemDump is documented as test-only; an error here means
		// something is fundamentally wrong with the wasmer memory
		// state. Return an empty slice and let the test assertion
		// surface the real failure.
		return []byte{}
	}
	return dump
}

// Id returns an identifier for the instance, unique at runtime.
//
// ISSUE-087: post the ISSUE-001 Phase 2 handle-API redesign, cgoInstance is
// no longer a real C heap pointer — it's an opaque small integer issued by
// the Rust handle_registry's monotonic counter, reinterpreted as a pointer
// for FFI wire compatibility. Formatting it as `%p` produced misleading
// "0x1", "0x2" pointer-shaped strings in logs. Decimal format is more
// honest for diagnostics. Map keys (instanceTracker.go) compare by string
// equality, so the format change has no semantic impact on those callers.
func (instance *Wasmer2Instance) ID() string {
	return fmt.Sprintf("%d", uintptr(unsafe.Pointer(instance.cgoInstance)))
}

// Reset resets the instance memories and globals.
//
// ISSUE-082: take instance.cleanMu only for the AlreadyClean READ at the
// top, then release before calling cWasmerInstanceReset. Holding the lock
// across the cgo call would risk deadlock against the Rust-side
// reentrant call paths (the same class of bug as the previous Mutex<Box<dyn
// InstanceLegacy>> attempt — see mx-vm-executor-rs ISSUE-001 history).
// TOCTOU between the unlock and cWasmerInstanceReset is tolerated because
// the Rust handle_registry is idempotent on operations against a destroyed
// instance.
func (instance *Wasmer2Instance) Reset() bool {
	instance.cleanMu.Lock()
	if instance.AlreadyClean {
		instance.cleanMu.Unlock()
		logWasmer2.Trace("reset: already cleaned instance", "id", instance.ID())
		return false
	}
	instance.cleanMu.Unlock()

	result := cWasmerInstanceReset(instance.cgoInstance)
	ok := result == cWasmerOk

	logWasmer2.Trace("reset: warm instance", "id", instance.ID(), "ok", ok)
	return ok
}

// IsInterfaceNil returns true if underlying object is nil
func (instance *Wasmer2Instance) IsInterfaceNil() bool {
	return instance == nil
}

// SetVMHooksPtr is a no-op on Wasmer2Instance.
//
// ISSUE-083: the wasmer2 architecture stores the vmHooks pointer at the
// EXECUTOR level (set via cWasmerExecutorContextDataSet in bridge2.go),
// not per-instance. Active wrappers and mocks keep these methods on the
// shared executor.Instance interface for compatibility, but wasmer2 instances
// themselves do not own this pointer.
//
// This method exists only to satisfy the executor.Instance interface.
// Callers that need to actually set the vmHooks pointer must operate on
// the executor directly. There is at least one skipped test
// (vmhost/hosttest/execution_test.go:322) that calls GetVMHooksPtr; for
// that reason this method is documented as no-op rather than panic. If
// the interface is later refactored to remove these methods from
// executor.Instance, this stub can go.
func (instance *Wasmer2Instance) SetVMHooksPtr(vmHooksPtr uintptr) {
	// intentionally empty — vmHooks pointer lives on the executor, not the
	// instance, in the wasmer2 architecture. See doc-comment above (ISSUE-083).
}

// GetVMHooksPtr returns 0 on Wasmer2Instance — see SetVMHooksPtr doc for
// rationale. Callers that need the actual pointer must read it from the
// executor, not the instance.
func (instance *Wasmer2Instance) GetVMHooksPtr() uintptr {
	return uintptr(0)
}
