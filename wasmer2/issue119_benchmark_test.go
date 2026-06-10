package wasmer2

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/multiversx/mx-chain-vm-go/executor"
)

var issue119Options = executor.CompilationOptions{}

func issue119ContractCode(b testing.TB) []byte {
	b.Helper()

	code, err := os.ReadFile(filepath.Join("..", "test", "contracts", "init-simple", "output", "init-simple.wasm"))
	if err != nil {
		b.Fatal(err)
	}

	return code
}

func issue119Executor(b testing.TB) *Wasmer2Executor {
	b.Helper()

	exec, err := CreateExecutor()
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(exec.Destroy)

	return exec
}

func issue119Instance(b testing.TB, exec *Wasmer2Executor, code []byte) *Wasmer2Instance {
	b.Helper()

	instance, err := exec.NewInstanceWithOptions(code, issue119Options)
	if err != nil {
		b.Fatal(err)
	}

	wasmerInstance, ok := instance.(*Wasmer2Instance)
	if !ok {
		b.Fatalf("expected *Wasmer2Instance, got %T", instance)
	}

	return wasmerInstance
}

func BenchmarkIssue119CreateDestroyExecutor(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		exec, err := CreateExecutor()
		if err != nil {
			b.Fatal(err)
		}
		exec.Destroy()
	}
}

func BenchmarkIssue119InstantiateAndClean(b *testing.B) {
	b.ReportAllocs()

	code := issue119ContractCode(b)
	exec := issue119Executor(b)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		instance := issue119Instance(b, exec, code)
		instance.Clean()
	}
}

func BenchmarkIssue119CacheRestoreAndClean(b *testing.B) {
	b.ReportAllocs()

	code := issue119ContractCode(b)
	exec := issue119Executor(b)
	instance := issue119Instance(b, exec, code)
	cache, err := instance.Cache()
	if err != nil {
		b.Fatal(err)
	}
	instance.Clean()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		restored, err := exec.NewInstanceFromCompiledCodeWithOptions(cache, issue119Options)
		if err != nil {
			b.Fatal(err)
		}
		restored.Clean()
	}
}

func BenchmarkIssue119HasFunction(b *testing.B) {
	b.ReportAllocs()

	code := issue119ContractCode(b)
	exec := issue119Executor(b)
	instance := issue119Instance(b, exec, code)
	b.Cleanup(func() { instance.Clean() })
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if !instance.HasFunction("init") {
			b.Fatal("expected init function")
		}
	}
}

func BenchmarkIssue119GetFunctionNames(b *testing.B) {
	b.ReportAllocs()

	code := issue119ContractCode(b)
	exec := issue119Executor(b)
	instance := issue119Instance(b, exec, code)
	b.Cleanup(func() { instance.Clean() })
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if len(instance.GetFunctionNames()) == 0 {
			b.Fatal("expected exported function names")
		}
	}
}

func BenchmarkIssue119ParallelInstantiateAndClean(b *testing.B) {
	b.ReportAllocs()

	code := issue119ContractCode(b)
	exec := issue119Executor(b)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			instance := issue119Instance(b, exec, code)
			instance.Clean()
		}
	})
}
