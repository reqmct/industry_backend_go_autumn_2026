package main

import "container/list"

type entry[K comparable, V any] struct {
	key   K
	value V
}
type LRUCache[K comparable, V any] struct {
	capacity int
	ll       list.List
	items    map[K]*list.Element
}

func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V] {
	return &LRUCache[K, V]{
		capacity: capacity,
		items:    make(map[K]*list.Element, capacity),
	}
}
func (c *LRUCache[K, V]) Get(key K) (value V, ok bool) {
	if c.capacity <= 0 {
		return
	}

	element, ok := c.items[key]
	if !ok {
		return value, false
	}

	c.ll.MoveToBack(element)

	return element.Value.(entry[K, V]).value, true
}
func (c *LRUCache[K, V]) Set(key K, value V) {
	if c.capacity <= 0 {
		return
	}

	element, ok := c.items[key]
	if ok {
		element.Value = entry[K, V]{
			key:   key,
			value: value,
		}
		return
	}

	if c.ll.Len() == c.capacity {
		front := c.ll.Front()
		frontEntry := front.Value.(entry[K, V])

		delete(c.items, frontEntry.key)
		c.ll.Remove(front)
	}

	newElement := c.ll.PushBack(entry[K, V]{
		key:   key,
		value: value,
	})

	c.items[key] = newElement
}

type LRU[K comparable, V any] interface {
	Get(K) (V, bool)
	Set(K, V)
}
