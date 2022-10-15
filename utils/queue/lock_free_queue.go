package queue

import (
	"sync/atomic"
	"unsafe"
)

type TaskQueue interface {
	Enqueue(interface{})
	Dequeue() interface{}
	IsEmpty() bool
}

type Queue struct {
	Head   unsafe.Pointer
	Tail   unsafe.Pointer
	Length int32
}

type node struct {
	Value interface{}
	Next  unsafe.Pointer
}

func NewQueue() TaskQueue {
	n := unsafe.Pointer(&node{})
	return &Queue{Head: n, Tail: n}
}

func (that *Queue) Enqueue(task interface{}) {
	n := &node{Value: task}
	for {
		tail := load(&that.Tail)
		next := load(&tail.Next)

		if tail == load(&that.Tail) {
			if next == nil {
				if cas(&tail.Next, next, n) {
					cas(&that.Tail, tail, n)
					atomic.AddInt32(&that.Length, 1)
					return
				}
			} else {
				cas(&that.Tail, tail, next)
			}
		}
	}
}

func (that *Queue) Dequeue() interface{} {
	for {
		head := load(&that.Head)
		tail := load(&that.Tail)
		next := load(&head.Next)
		if head == load(&that.Head) {
			if head == tail {
				if next == nil {
					return nil
				}
				cas(&that.Tail, tail, next)
			} else {
				task := next.Value // the first node is blank.
				if cas(&that.Head, head, next) {
					atomic.AddInt32(&that.Length, -1)
					return task
				}
			}
		}
	}
}

func (q *Queue) IsEmpty() bool {
	return atomic.LoadInt32(&q.Length) == 0
}

func load(p *unsafe.Pointer) (n *node) {
	return (*node)(atomic.LoadPointer(p))
}

func cas(p *unsafe.Pointer, old, new *node) bool {
	return atomic.CompareAndSwapPointer(p, unsafe.Pointer(old), unsafe.Pointer(new))
}
