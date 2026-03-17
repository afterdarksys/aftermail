package plugins

import (
	"fmt"
	"log"
	"path/filepath"
	"plugin"
	"sync"
)

// Plugin defines the interface all AfterMail plugins must implement.
type Plugin interface {
	// Name returns the unique name of the plugin.
	Name() string
	// Description returns a short description.
	Description() string
	// Init is called when the plugin is loaded.
	Init() error
	// Shutdown is called when the application closes.
	Shutdown() error
}

// Manager handles the loading and lifecycle of Go plugins.
type Manager struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
	dir     string
}

// NewManager creates a new plugin manager looking for .so files in the specified directory.
func NewManager(pluginDir string) *Manager {
	return &Manager{
		plugins: make(map[string]Plugin),
		dir:     pluginDir,
	}
}

// LoadPlugins scans the plugin directory and loads all valid .so modules.
func (m *Manager) LoadPlugins() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	files, err := filepath.Glob(filepath.Join(m.dir, "*.so"))
	if err != nil {
		return fmt.Errorf("failed to scan plugin directory: %w", err)
	}

	for _, file := range files {
		log.Printf("[Plugins] Loading dynamic library: %s", file)
		p, err := plugin.Open(file)
		if err != nil {
			log.Printf("[Plugins] Warning: Failed to open %s: %v", file, err)
			continue
		}

		// Look for the exported symbol 'AfterMailPlugin'
		sym, err := p.Lookup("AfterMailPlugin")
		if err != nil {
			log.Printf("[Plugins] Warning: Plugin %s missing 'AfterMailPlugin' symbol", file)
			continue
		}

		// Assert it matches our interface
		ampPlugin, ok := sym.(Plugin)
		if !ok {
			log.Printf("[Plugins] Warning: Symbol in %s does not implement plugins.Plugin interface", file)
			continue
		}

		// Initialize
		if err := ampPlugin.Init(); err != nil {
			log.Printf("[Plugins] Warning: Plugin %s failed to initialize: %v", ampPlugin.Name(), err)
			continue
		}

		log.Printf("[Plugins] Successfully registered plugin: %s (%s)", ampPlugin.Name(), ampPlugin.Description())
		m.plugins[ampPlugin.Name()] = ampPlugin
	}

	return nil
}

// GetLoaded returns a list of all successfully loaded plugins.
func (m *Manager) GetLoaded() []Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []Plugin
	for _, p := range m.plugins {
		list = append(list, p)
	}
	return list
}

// ShutdownAll gracefully terminates all loaded plugins.
func (m *Manager) ShutdownAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, p := range m.plugins {
		if err := p.Shutdown(); err != nil {
			log.Printf("[Plugins] Error shutting down plugin %s: %v", name, err)
		}
	}
}
