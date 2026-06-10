# VM Hooks Code Generator

The code generator produces boilerplate for both this Go VM and the Rust
executor repository.

Generated Go files include:

- `executor/vmHooks.go`
- `executor/wrapper/wrapperVMHooks.go`
- `wasmer2/wasmer2ImportsCgo.go`
- `wasmer2/wasmer2Names.go`
- `wasmer2/opcodeCost.go`
- `mock/context/executorMockFunc.go`

Generated Rust files include:

- `vm-executor/src/vm_hooks.rs`
- `vm-executor/src/new_traits/vm_hooks_new.rs`
- `vm-executor/src/new_traits/vm_hooks_legacy_adapter.rs`
- `vm-executor/src/opcode_cost.rs`
- `c-api/src/capi_vm_hooks.rs`
- `c-api/src/capi_vm_hook_pointers.rs`
- `vm-executor-wasmer/src/wasmer_imports.rs`
- `vm-executor-experimental/src/we_imports.rs`
- `vm-executor-wasmer/src/wasmer_metering_helpers.rs`

## Regeneration

To regenerate only the Go-side outputs and the intermediate Rust outputs, run:

```bash
go generate ./vmhost/vmhooks
```

To also copy generated Rust files into a local `mx-vm-executor-rs` checkout,
create `vmhost/vmhooks/generate/cmd/wasm-vm-executor-rs-path.txt` containing
the absolute path to that checkout, then run the same `go generate` command.

After Rust files are regenerated, rebuild the Rust C header from
`mx-vm-executor-rs` so the Go bridge and Rust dynamic library stay in lockstep.
The generated header in:

```text
mx-vm-executor-rs/c-api/libvmexeccapi.h
```

must be copied to:

```text
mx-chain-vm-go/wasmer2/libvmexeccapi.h
```

## ABI Version Discipline

The Go bridge checks the Rust dynamic library ABI at startup through
`vm_exec_api_version()`. When any C ABI or semantic FFI contract changes:

1. Bump `VM_EXEC_API_VERSION` in `mx-vm-executor-rs/c-api/src/lib.rs`.
2. Rebuild the Rust dynamic library and `c-api/libvmexeccapi.h`.
3. Copy the refreshed `libvmexeccapi.h` into `mx-chain-vm-go/wasmer2`.
4. Bump `expectedAPIVersion` in `mx-chain-vm-go/wasmer2/wasmer2Executor.go`.
5. Run focused Wasmer2 tests.

## CI Guardrails

`wasmer2/abiVersion_test.go` verifies that:

- the linked Rust library reports the ABI version expected by the Go bridge,
- the local `wasmer2/libvmexeccapi.h` declares the same ABI version,
- if `MX_VM_EXECUTOR_RS_PATH` is set, the Go-side header is byte-for-byte
  identical to `mx-vm-executor-rs/c-api/libvmexeccapi.h`.

For cross-repo CI, checkout `mx-vm-executor-rs` beside this repository and run
the Wasmer2 tests with `MX_VM_EXECUTOR_RS_PATH` pointing to that checkout.

## Review Checklist For New Hooks

Every new hook must be reviewed across:

- Go VM hook interface and implementation,
- Go Wasmer2 cgo export,
- Go Wasmer2 import/name list,
- Rust VM hook trait,
- Rust C function-pointer struct,
- Rust Wasmer import registration,
- gas schedule and feature flag behavior,
- positive and negative tests.

Do not hand-edit generated files unless the same change is also added to the
generator.
