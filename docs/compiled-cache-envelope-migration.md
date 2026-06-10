# Compiled Cache Envelope Migration

NewArc VM executor builds wrap Wasmer compiled-cache artifacts in a small
validation envelope before they are stored by the node. The envelope is checked
when a cached artifact is restored.

The envelope rejects artifacts that do not match the VM build or execution
configuration. This is intentional: stale or corrupted compiled machine code
must not be handed directly to Wasmer.

## What Changes During Upgrade

Compiled cache entries produced before the envelope change are legacy raw
Wasmer artifacts. They do not carry the NewArc envelope and will be rejected on
first restore after upgrade.

This does not make smart contract execution fail in the normal node path. When
restoring from compiled cache fails, the VM host logs the cache restore error
and falls back to compiling from the original contract bytecode. The regenerated
compiled artifact is then stored with the envelope.

Expected operational effects:

- the first execution of affected contracts after upgrade may be slower,
- compiled-cache hit rate may temporarily drop,
- old raw cache bytes are naturally replaced as contracts are executed again.

## Rejection Messages

The Rust VM executor records the exact envelope rejection reason as the C API
last error. The Go bridge wraps that message into the restore error logged by
`vmhost/contexts/runtime.go`.

Use the message to classify the event:

- `compiled code was not found`: normal cache miss; bytecode compilation follows.
- `compiled cache envelope is too short`: legacy raw artifact, truncated bytes,
  or otherwise non-envelope data.
- `compiled cache envelope magic mismatch`: legacy raw artifact large enough to
  pass the minimum length check, or non-envelope data.
- `compiled cache envelope version mismatch`: cache was produced by a different
  envelope format version.
- `compiled cache envelope compilation options mismatch`: cache was produced
  with different metering/runtime-breakpoint/memory-grow options.
- `compiled cache envelope payload length mismatch`: stored bytes are truncated
  or have trailing data.
- `compiled cache envelope hash mismatch`: payload or envelope metadata changed
  after the artifact was created, or the VM package/options fingerprint differs.

For all of the above restore rejections, the expected node behavior is fallback
to bytecode compilation where the VM host has the original contract bytecode.

## Operator Runbook

Before upgrading validators or observers to a NewArc build with cache-envelope
support:

1. Plan for a temporary increase in contract compilation work after restart.
2. Keep the original contract bytecode database intact. The compiled cache is
   disposable; bytecode is not.
3. If startup logs are noisy or disk pressure is a concern, clear only the VM
   compiled-code cache directory for the node data path before restart.
4. Do not copy compiled-cache directories between nodes running different VM
   executor builds or gas schedule settings.
5. Monitor logs for repeated `compiled cache envelope ...` messages. Repeated
   messages for the same contracts usually mean old cache data is still present;
   repeated `compilation options mismatch` after cleanup points to inconsistent
   gas schedule/runtime options across nodes.

The exact cache directory is deployment-specific. Confirm it from the node data
directory and storage configuration before deleting anything. When unsure,
prefer moving the compiled cache directory aside rather than deleting the full
node database.

## Rollback Notes

A rollback to a build without envelope support cannot read NewArc enveloped
compiled artifacts as raw Wasmer cache bytes. Clear or move the compiled-code
cache directory before starting the older build.

The chain state and smart contract bytecode are unaffected by this cache format
change.
