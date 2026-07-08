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

## C ABI Discipline

The Go bridge must remain byte-layout compatible with
`mx-vm-executor-rs/c-api/libvmexeccapi.h`. The clean upstream C API does not
export a runtime ABI-version function, so ABI safety is enforced by keeping the
generated Go bridge, the checked-in header, and the rebuilt dynamic library in
lockstep. When any C ABI or semantic FFI contract changes:

1. Rebuild the Rust dynamic library and `c-api/libvmexeccapi.h`.
2. Copy the refreshed `libvmexeccapi.h` into `mx-chain-vm-go/wasmer2`.
3. Refresh the platform dynamic libraries in `mx-chain-vm-go/wasmer2`.
4. Run focused Wasmer2 and hook-generation tests.

## CI Guardrails

The Wasmer2 and hook-generation tests verify that:

- the Go-side hook pointer struct still matches the Rust C header layout used by
  the checked-in bridge code,
- DRWA and official hook imports are registered in both Go and Rust surfaces,
- scenario execution reaches the DRWA native hook paths.

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
