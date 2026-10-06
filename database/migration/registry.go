package migration

import (
	"fmt"
	"sort"
	"sync"
)

var (
	registryMu sync.RWMutex
	registry   = make(map[string]Migration)
)

func Register(
	name string,
	up BlueprintFactory,
	down BlueprintFactory,
) error {

	if name == "" {
		return fmt.Errorf(
			"copytygo: migration name cannot be empty",
		)
	}

	if up == nil {
		return fmt.Errorf(
			"copytygo: migration %q requires an Up function",
			name,
		)
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	if _, exists := registry[name]; exists {
		return fmt.Errorf(
			"copytygo: migration %q already registered",
			name,
		)
	}

	registry[name] = Migration{
		Name: name,
		Up:   up,
		Down: down,
	}

	return nil
}

func All() []Migration {
	registryMu.RLock()
	defer registryMu.RUnlock()

	names := make([]string, 0, len(registry))

	for name := range registry {
		names = append(names, name)
	}

	sort.Strings(names)

	migrations := make(
		[]Migration,
		0,
		len(names),
	)

	for _, name := range names {
		migrations = append(
			migrations,
			registry[name],
		)
	}

	return migrations
}

func Find(name string) (Migration, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	item, exists := registry[name]

	return item, exists
}


func ResetRegistry() {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = make(map[string]Migration)
}
