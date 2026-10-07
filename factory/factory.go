package factory

import (
	"fmt"
	"sync/atomic"
)

type Factory[T any] struct {
	sequence atomic.Uint64
	make     func(sequence uint64) T
}

func New[T any](make func(sequence uint64) T) *Factory[T] {
	return &Factory[T]{make: make}
}

func (f *Factory[T]) Make() (T, error) {
	var zero T
	if f == nil || f.make == nil {
		return zero, fmt.Errorf("copytygo factory: generator is required")
	}
	return f.make(f.sequence.Add(1)), nil
}

func (f *Factory[T]) Count(count int) ([]T, error) {
	if count < 0 {
		return nil, fmt.Errorf("copytygo factory: count cannot be negative")
	}
	items := make([]T, 0, count)
	for i := 0; i < count; i++ {
		item, err := f.Make()
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}
