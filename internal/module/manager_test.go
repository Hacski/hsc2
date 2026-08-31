package module

import (
	"context"
	"sync"
	"testing"
)

func TestLoadUnloadSwap(t *testing.T) {
	m := NewManager()
	m.Register("alpha", func() Module { return &fakeMod{name: "alpha", version: "1.0.0"} }, true)
	m.Register("beta", func() Module { return &fakeMod{name: "beta", version: "1.0.0"} }, false)

	ctx := context.Background()
	if err := m.AutoLoad(ctx); err != nil {
		t.Fatal(err)
	}
	if !m.IsLoaded("alpha") {
		t.Fatal("alpha must autoload")
	}
	if m.IsLoaded("beta") {
		t.Fatal("beta must not autoload")
	}

	if err := m.Load(ctx, "beta"); err != nil {
		t.Fatal(err)
	}
	out, err := m.Execute(ctx, "beta", Context{Operator: "op1"})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "ran beta by op1" {
		t.Fatalf("unexpected output %q", out)
	}

	// hot swap: unload alpha then load it back without redeploying server
	if err := m.Swap(ctx, "alpha"); err != nil {
		t.Fatal(err)
	}
	if m.IsLoaded("alpha") {
		t.Fatal("alpha should be unloaded after swap")
	}
	if err := m.Swap(ctx, "alpha"); err != nil {
		t.Fatal(err)
	}
	if !m.IsLoaded("alpha") {
		t.Fatal("alpha should be reloaded after second swap")
	}

	info := m.Info()
	if len(info) != 2 {
		t.Fatalf("expected 2 registered modules, got %d", len(info))
	}
}

func TestConcurrentHotSwap(t *testing.T) {
	m := NewManager()
	m.Register("conc", func() Module { return &fakeMod{name: "conc", version: "1"} }, true)
	ctx := context.Background()
	m.AutoLoad(ctx)
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.Swap(ctx, "conc")
			m.Execute(ctx, "conc", Context{})
		}()
	}
	wg.Wait()
}
