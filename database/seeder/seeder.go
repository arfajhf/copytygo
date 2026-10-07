package seeder

import (
	"context"
	"fmt"
)

type Seeder interface {
	Name() string
	Run(context.Context) error
}

type Func struct {
	SeederName string
	Handler    func(context.Context) error
}

func (f Func) Name() string { return f.SeederName }

func (f Func) Run(ctx context.Context) error {
	if f.Handler == nil {
		return fmt.Errorf("copytygo seeder: %s has no handler", f.SeederName)
	}
	return f.Handler(ctx)
}

type Registry struct {
	items []Seeder
}

func New() *Registry {
	return &Registry{}
}

func (r *Registry) Add(seed ...Seeder) {
	r.items = append(r.items, seed...)
}

func (r *Registry) Run(ctx context.Context) error {
	for _, seed := range r.items {
		if seed == nil {
			continue
		}
		if err := seed.Run(ctx); err != nil {
			return fmt.Errorf("copytygo seeder %q: %w", seed.Name(), err)
		}
	}
	return nil
}

func (r *Registry) Items() []Seeder {
	return append([]Seeder(nil), r.items...)
}
