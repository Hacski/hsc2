package module

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

type Manager struct {
	mu        sync.RWMutex
	available map[string]Factory
	loaded    map[string]Module
	flags     map[string]map[string]bool
	locks     map[string]*sync.Mutex
}

type Factory func() Module

func NewManager() *Manager {
	return &Manager{
		available: map[string]Factory{},
		loaded:    map[string]Module{},
		flags:     map[string]map[string]bool{},
		locks:     map[string]*sync.Mutex{},
	}
}

func (m *Manager) lockFor(name string) *sync.Mutex {
	m.mu.Lock()
	defer m.mu.Unlock()
	l, ok := m.locks[name]
	if !ok {
		l = &sync.Mutex{}
		m.locks[name] = l
	}
	return l
}

func (m *Manager) Register(name string, f Factory, autoLoad bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.available[name] = f
	if m.flags[name] == nil {
		m.flags[name] = map[string]bool{}
	}
	m.flags[name]["auto"] = autoLoad
}

func (m *Manager) Load(ctx context.Context, name string) error {
	m.mu.Lock()
	if _, ok := m.loaded[name]; ok {
		m.mu.Unlock()
		return fmt.Errorf("module %s already loaded", name)
	}
	f, ok := m.available[name]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("module %s not registered", name)
	}
	mod := f()
	m.loaded[name] = mod
	m.mu.Unlock()

	if err := mod.OnLoad(ctx); err != nil {
		m.mu.Lock()
		delete(m.loaded, name)
		m.mu.Unlock()
		return err
	}
	return nil
}

func (m *Manager) Unload(ctx context.Context, name string) error {
	m.mu.Lock()
	mod, ok := m.loaded[name]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("module %s not loaded", name)
	}
	m.mu.Unlock()
	if err := mod.OnUnload(ctx); err != nil {
		return err
	}
	m.mu.Lock()
	delete(m.loaded, name)
	m.mu.Unlock()
	return nil
}

func (m *Manager) IsLoaded(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.loaded[name]
	return ok
}

func (m *Manager) Execute(ctx context.Context, name string, c Context) ([]byte, error) {
	m.mu.RLock()
	mod, ok := m.loaded[name]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("module %s not loaded", name)
	}
	return mod.Execute(ctx, c)
}

func (m *Manager) Swap(ctx context.Context, name string) error {
	lock := m.lockFor(name)
	lock.Lock()
	defer lock.Unlock()
	if m.IsLoaded(name) {
		return m.Unload(ctx, name)
	}
	return m.Load(ctx, name)
}

func (m *Manager) AutoLoad(ctx context.Context) error {
	m.mu.RLock()
	names := []string{}
	for name, f := range m.available {
		if m.flags[name] != nil && m.flags[name]["auto"] {
			names = append(names, name)
		}
		_ = f
	}
	m.mu.RUnlock()
	sort.Strings(names)
	for _, n := range names {
		if err := m.Load(ctx, n); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) Info() []Info {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []Info{}
	for name, f := range m.available {
		mod := f()
		_, loaded := m.loaded[name]
		auto := m.flags[name] != nil && m.flags[name]["auto"]
		out = append(out, Info{Name: name, Version: mod.Version(), Loaded: loaded, AutoLoad: auto})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
