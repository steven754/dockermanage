// Package broker 提供轻量级发布订阅，供 SSE 广播使用。
package broker

import "sync"

// Broker 泛型发布订阅中心。
//
// 订阅者通过 Subscribe 获得一个只读 channel，调用方负责在不再需要时
// 触发返回的取消函数，避免 channel 泄漏。
type Broker[T any] struct {
	mu     sync.RWMutex
	subs   map[int]chan T
	nextID int
}

// New 创建 Broker
func New[T any]() *Broker[T] {
	return &Broker[T]{subs: make(map[int]chan T)}
}

// Subscribe 订阅消息，返回消息 channel 与取消订阅函数。
//
// buffer 为消息缓冲长度；缓冲满时丢弃最旧消息，保证生产者不被阻塞。
func (b *Broker[T]) Subscribe(buffer int) (<-chan T, func()) {
	if buffer <= 0 {
		buffer = 16
	}
	ch := make(chan T, buffer)

	b.mu.Lock()
	id := b.nextID
	b.nextID++
	b.subs[id] = ch
	b.mu.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			b.mu.Lock()
			if c, ok := b.subs[id]; ok {
				delete(b.subs, id)
				close(c)
			}
			b.mu.Unlock()
		})
	}
	return ch, cancel
}

// Publish 向所有订阅者广播消息，慢订阅者会丢失最旧消息。
func (b *Broker[T]) Publish(v T) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subs {
		select {
		case ch <- v:
		default:
			// 缓冲已满，丢弃最旧的一条后重试
			select {
			case <-ch:
			default:
			}
			select {
			case ch <- v:
			default:
			}
		}
	}
}

// SubscriberCount 返回当前订阅者数量
func (b *Broker[T]) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subs)
}
