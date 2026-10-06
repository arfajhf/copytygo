package core

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
)

type memoryRecord map[string]any

type memoryBucket struct {
	nextID int64
	items  map[string]memoryRecord
}

var memoryDB = struct {
	sync.RWMutex
	buckets map[string]*memoryBucket
}{
	buckets: make(map[string]*memoryBucket),
}

func memoryBucketFor(name string) *memoryBucket {
	memoryDB.Lock()
	defer memoryDB.Unlock()

	bucket, ok := memoryDB.buckets[name]
	if !ok {
		bucket = &memoryBucket{
			nextID: 1,
			items:  make(map[string]memoryRecord),
		}
		memoryDB.buckets[name] = bucket
	}
	return bucket
}

func (ctx *Context) MemoryIndex(resource string) error {
	bucket := memoryBucketFor(resource)

	memoryDB.RLock()
	defer memoryDB.RUnlock()

	data := make([]Map, 0, len(bucket.items))
	for id, item := range bucket.items {
		row := Map{"id": id}
		for key, value := range item {
			row[key] = value
		}
		data = append(data, row)
	}

	return ctx.JSON(Map{"data": data})
}

func (ctx *Context) MemoryShow(resource, id string) error {
	bucket := memoryBucketFor(resource)

	memoryDB.RLock()
	item, ok := bucket.items[id]
	memoryDB.RUnlock()
	if !ok {
		return ctx.NotFound("Resource not found")
	}

	row := Map{"id": id}
	for key, value := range item {
		row[key] = value
	}
	return ctx.JSON(row)
}

func (ctx *Context) MemoryStore(resource string, data Map) error {
	bucket := memoryBucketFor(resource)

	memoryDB.Lock()
	id := strconv.FormatInt(bucket.nextID, 10)
	bucket.nextID++
	record := memoryRecord{}
	for key, value := range data {
		record[key] = value
	}
	bucket.items[id] = record
	memoryDB.Unlock()

	row := Map{"id": id}
	for key, value := range record {
		row[key] = value
	}
	return ctx.Status(http.StatusCreated).JSON(row)
}

func (ctx *Context) MemoryUpdate(resource, id string, data Map) error {
	bucket := memoryBucketFor(resource)

	memoryDB.Lock()
	record, ok := bucket.items[id]
	if !ok {
		memoryDB.Unlock()
		return ctx.NotFound("Resource not found")
	}
	for key, value := range data {
		record[key] = value
	}
	bucket.items[id] = record
	memoryDB.Unlock()

	row := Map{"id": id}
	for key, value := range record {
		row[key] = value
	}
	return ctx.JSON(row)
}

func (ctx *Context) MemoryDestroy(resource, id string) error {
	bucket := memoryBucketFor(resource)

	memoryDB.Lock()
	if _, ok := bucket.items[id]; !ok {
		memoryDB.Unlock()
		return ctx.NotFound("Resource not found")
	}
	delete(bucket.items, id)
	memoryDB.Unlock()

	return ctx.NoContent(http.StatusNoContent)
}

func ResetMemory(resource string) {
	memoryDB.Lock()
	defer memoryDB.Unlock()
	if resource == "" {
		memoryDB.buckets = make(map[string]*memoryBucket)
		return
	}
	delete(memoryDB.buckets, resource)
}

func MemoryCount(resource string) int {
	bucket := memoryBucketFor(resource)
	memoryDB.RLock()
	defer memoryDB.RUnlock()
	return len(bucket.items)
}

func MemoryDebug(resource string) string {
	return fmt.Sprintf("%s:%d", resource, MemoryCount(resource))
}
