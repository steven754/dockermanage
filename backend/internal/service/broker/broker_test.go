package broker_test

import (
	"testing"
	"time"

	"github.com/steven754/dockermanage/backend/internal/service/broker"
)

func TestBrokerPublishSubscribe(t *testing.T) {
	b := broker.New[string]()

	ch, cancel := b.Subscribe(4)
	defer cancel()

	b.Publish("hello")

	select {
	case got := <-ch:
		if got != "hello" {
			t.Fatalf("期望 hello，实际 %s", got)
		}
	case <-time.After(time.Second):
		t.Fatal("未收到广播消息")
	}
}

func TestBrokerMultipleSubscribers(t *testing.T) {
	b := broker.New[int]()

	ch1, cancel1 := b.Subscribe(2)
	defer cancel1()
	ch2, cancel2 := b.Subscribe(2)
	defer cancel2()

	if b.SubscriberCount() != 2 {
		t.Fatalf("期望 2 个订阅者，实际 %d", b.SubscriberCount())
	}

	b.Publish(42)

	for i, ch := range []<-chan int{ch1, ch2} {
		select {
		case got := <-ch:
			if got != 42 {
				t.Fatalf("订阅者 %d 期望 42，实际 %d", i, got)
			}
		case <-time.After(time.Second):
			t.Fatalf("订阅者 %d 未收到消息", i)
		}
	}
}

func TestBrokerCancelUnsubscribes(t *testing.T) {
	b := broker.New[string]()

	_, cancel := b.Subscribe(1)
	if b.SubscriberCount() != 1 {
		t.Fatalf("期望 1 个订阅者，实际 %d", b.SubscriberCount())
	}

	cancel()
	if b.SubscriberCount() != 0 {
		t.Fatalf("取消后期望 0 个订阅者，实际 %d", b.SubscriberCount())
	}

	// 重复取消不应 panic
	cancel()
}

func TestBrokerDropsOldestWhenBufferFull(t *testing.T) {
	b := broker.New[int]()

	ch, cancel := b.Subscribe(1)
	defer cancel()

	b.Publish(1)
	b.Publish(2)

	// 缓冲为 1，第二次发布会丢弃第一条
	got := <-ch
	if got != 2 {
		t.Fatalf("期望保留最新消息 2，实际 %d", got)
	}
}

func TestBrokerPublishAfterCancel(t *testing.T) {
	b := broker.New[string]()

	_, cancel := b.Subscribe(1)
	cancel()

	// 取消后发布不应 panic
	b.Publish("after-cancel")
}
