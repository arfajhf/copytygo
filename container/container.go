package container

import (
	"fmt"
	"reflect"
	"sync"
)

type Factory func(*Container) (any, error)

type binding struct {
	factory   Factory
	singleton bool
	instance  any
	resolved  bool
}

type Container struct {
	mu       sync.RWMutex
	bindings map[string]*binding
}

func New() *Container {
	return &Container{bindings: make(map[string]*binding)}
}

var Default = New()

func (c *Container) Bind(name string, factory Factory) {
	c.set(name, factory, false)
}

func (c *Container) Singleton(name string, factory Factory) {
	c.set(name, factory, true)
}

func (c *Container) Instance(name string, value any) {
	c.mu.Lock()
	c.bindings[name] = &binding{
		singleton: true,
		instance:  value,
		resolved:  true,
	}
	c.mu.Unlock()
}

func (c *Container) Has(name string) bool {
	c.mu.RLock()
	_, ok := c.bindings[name]
	c.mu.RUnlock()
	return ok
}

func (c *Container) Resolve(name string) (any, error) {
	c.mu.RLock()
	item, ok := c.bindings[name]
	if !ok {
		c.mu.RUnlock()
		return nil, fmt.Errorf("copytygo container: binding %q not found", name)
	}
	if item.singleton && item.resolved {
		value := item.instance
		c.mu.RUnlock()
		return value, nil
	}
	factory := item.factory
	singleton := item.singleton
	c.mu.RUnlock()

	if factory == nil {
		return nil, fmt.Errorf("copytygo container: binding %q has no factory", name)
	}

	value, err := factory(c)
	if err != nil {
		return nil, fmt.Errorf("copytygo container: resolve %q: %w", name, err)
	}

	if singleton {
		c.mu.Lock()
		if current, exists := c.bindings[name]; exists {
			if current.resolved {
				value = current.instance
			} else {
				current.instance = value
				current.resolved = true
			}
		}
		c.mu.Unlock()
	}

	return value, nil
}

func ResolveAs[T any](c *Container, name string) (T, error) {
	var zero T
	value, err := c.Resolve(name)
	if err != nil {
		return zero, err
	}
	typed, ok := value.(T)
	if !ok {
		return zero, fmt.Errorf(
			"copytygo container: binding %q resolved to %s, expected %s",
			name,
			typeName(value),
			typeName(zero),
		)
	}
	return typed, nil
}

func (c *Container) Forget(name string) {
	c.mu.Lock()
	delete(c.bindings, name)
	c.mu.Unlock()
}

func (c *Container) Flush() {
	c.mu.Lock()
	c.bindings = make(map[string]*binding)
	c.mu.Unlock()
}

func (c *Container) set(name string, factory Factory, singleton bool) {
	c.mu.Lock()
	c.bindings[name] = &binding{factory: factory, singleton: singleton}
	c.mu.Unlock()
}

func typeName(value any) string {
	if value == nil {
		return "<nil>"
	}
	return reflect.TypeOf(value).String()
}
