package hw04lrucache

import "sync"

const unexpectedTypePanicMessage = "Unexpected type in Cache"

type Key string

type Cache interface {
	Set(key Key, value interface{}) bool
	Get(key Key) (interface{}, bool)
	Clear()
}

type lruCache struct {
	capacity int
	queue    List
	items    map[Key]*ListItem

	mu *sync.Mutex
}

type cacheItem struct {
	value interface{}
	key   Key
}

func (l *lruCache) Set(key Key, value interface{}) bool {
	// т.к. Set, Get и Clear выполняются за константное время,
	// то расставлять локи в конкретных местах функции не имеет смысла
	// по крайней мере я так думаю
	defer l.mu.Unlock()
	l.mu.Lock()
	if item, ok := l.items[key]; ok {
		valueCacheItem, ok := item.Value.(*cacheItem)
		if !ok {
			// спорный момент с паникой, вроде из библиотек и не прикольно паниковать, но и такого кейса быть
			// не должно. Все же решил добавить обработку для читаемости кода
			panic(unexpectedTypePanicMessage)
		}
		valueCacheItem.value = value
		l.queue.MoveToFront(item)
		return true
	}

	if l.queue.Len() == l.capacity {
		cacheItemToDelete, ok := l.queue.Back().Value.(*cacheItem)
		if !ok {
			// аналогично
			panic(unexpectedTypePanicMessage)
		}
		delete(l.items, cacheItemToDelete.key)
		l.queue.Remove(l.queue.Back())
	}

	l.items[key] = l.queue.PushFront(&cacheItem{
		value: value,
		key:   key,
	})

	return false
}

func (l *lruCache) Get(key Key) (interface{}, bool) {
	defer l.mu.Unlock()
	l.mu.Lock()
	if item, ok := l.items[key]; ok {
		l.queue.MoveToFront(item)
		cacheItemToReturn, ok := item.Value.(*cacheItem)
		if !ok {
			// аналогично методу Set
			panic(unexpectedTypePanicMessage)
		}
		return cacheItemToReturn.value, true
	}
	return nil, false
}

func (l *lruCache) Clear() {
	defer l.mu.Unlock()
	l.mu.Lock()
	l.queue = NewList()
	l.items = make(map[Key]*ListItem, l.capacity)
}

func NewCache(capacity int) Cache {
	return &lruCache{
		capacity: capacity,
		queue:    NewList(),
		items:    make(map[Key]*ListItem, capacity),

		mu: &sync.Mutex{},
	}
}
