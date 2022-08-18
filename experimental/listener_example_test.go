package experimental_test

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"sort"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	. "github.com/tetratelabs/wazero/experimental"
	"github.com/tetratelabs/wazero/wasi_snapshot_preview1"
)

// listenerWasm was generated by the following:
//
//	cd testdata; wat2wasm --debug-names listener.wat
//
//go:embed logging/testdata/listener.wasm
var listenerWasm []byte

// uniqGoFuncs implements both FunctionListenerFactory and FunctionListener
type uniqGoFuncs map[string]struct{}

// callees returns the go functions called.
func (u uniqGoFuncs) callees() []string {
	ret := make([]string, 0, len(u))
	for k := range u {
		ret = append(ret, k)
	}
	// Sort names for consistent iteration
	sort.Strings(ret)
	return ret
}

// NewListener implements FunctionListenerFactory.NewListener
func (u uniqGoFuncs) NewListener(def api.FunctionDefinition) FunctionListener {
	if def.GoFunc() == nil {
		return nil // only track go funcs
	}
	return u
}

// Before implements FunctionListener.Before
func (u uniqGoFuncs) Before(ctx context.Context, def api.FunctionDefinition, _ []uint64) context.Context {
	u[def.DebugName()] = struct{}{}
	return ctx
}

// After implements FunctionListener.After
func (u uniqGoFuncs) After(context.Context, api.FunctionDefinition, error, []uint64) {}

// This shows how to make a listener that counts go function calls.
func Example_customListenerFactory() {
	u := uniqGoFuncs{}

	// Set context to one that has an experimental listener
	ctx := context.WithValue(context.Background(), FunctionListenerFactoryKey{}, u)

	r := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigInterpreter())
	defer r.Close(ctx) // This closes everything this Runtime created.

	if _, err := wasi_snapshot_preview1.Instantiate(ctx, r); err != nil {
		log.Panicln(err)
	}

	// Compile the WebAssembly module using the default configuration.
	code, err := r.CompileModule(ctx, listenerWasm, wazero.NewCompileConfig())
	if err != nil {
		log.Panicln(err)
	}

	mod, err := r.InstantiateModule(ctx, code, wazero.NewModuleConfig())
	if err != nil {
		log.Panicln(err)
	}

	for i := 0; i < 5; i++ {
		if _, err = mod.ExportedFunction("rand").Call(ctx, 4); err != nil {
			log.Panicln(err)
		}
	}

	// A Go function was called multiple times, but we should only see it once.
	for _, f := range u.callees() {
		fmt.Println(f)
	}

	// Output:
	// wasi_snapshot_preview1.random_get
}
