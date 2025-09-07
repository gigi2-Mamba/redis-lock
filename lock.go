package redis_lock

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"time"
)

// distribute mutex
/*
 提供一个中间件分布式锁，利用redis setNx实现，  set  if not exist
*/

var (
	// You only need to think how to import lua?  you need declare the string first, then //go:embed  file url
	//go:embed  script/unlock.lua
	luaUnlock string
	//go:embed script/refresh.lua
	luaRefresh             string
	ErrLockNotHold         = errors.New("Not holding the lock")
	ErrFailedToPreemptLock = errors.New("preempt fail")
)

// no need to change
// how to mock  and integration test
type Client struct {
	client redis.Cmdable
}
type Lock struct {
	client     redis.Cmdable
	key        string
	value      string
	expiration time.Duration //why not use the time unit directly?
	unlock     chan struct{}
}

func NewClient(c redis.Cmdable) *Client {
	return &Client{
		client: c,
	}
}

func NewLock(c redis.Cmdable, key, value string, expiration time.Duration) *Lock {
	return &Lock{
		client:     c,
		key:        key,
		value:      value,
		expiration: expiration,
		unlock:     make(chan struct{}, 1),
	}
}

// notice
func (l *Lock) AutoRefresh(interval, timeout time.Duration) {
	// go惯用都是用一个context,但是没感觉这里context有啥用
	// 该怎么续约呢？ 每隔多少秒续约一次
	ticker := time.NewTicker(interval)
	// 续约要控制时间，否则会一直续约。 每次续约要控制时间间隔，
	// 规定时间内没有续完成结束
	// 续约那肯定是一直的事情，for loop
	refreshCh := make(chan struct{}, 1)
	defer close(refreshCh)
	//利用带有一个buffer的
	for {
		// 有什么分支呢？ 复杂的东西可能产生嵌套，嵌套是个好技能
		select {
		case <-refreshCh:
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			err := l.Refresh(ctx)
			// 只考虑超时错误
			if err == context.DeadlineExceeded {
				refreshCh <- struct{}{}
			}
			cancel()
		case <-ticker.C:
			// refresh 有时间限制
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			err := l.Refresh(ctx)
			// 只考虑超时错误
			if err == context.DeadlineExceeded {
				refreshCh <- struct{}{}
			}
			cancel()

		case <-l.unlock:
			return
		}
	}

}
func (l *Lock) Refresh(ctx context.Context) error {
	res, err := l.client.Eval(ctx, luaRefresh, []string{l.key}, l.value, l.expiration.Milliseconds()).Int64()
	if err == redis.Nil {
		return ErrLockNotHold
	}
	if err != nil {
		return err
	}
	if res != 1 {
		return ErrLockNotHold
	}
	return nil
}

func (c *Client) TryLock(ctx context.Context, key string, expiration time.Duration) (lock *Lock, err error) {
	value := uuid.New().String() // unique the value
	res, err := c.client.SetNX(ctx, key, value, time.Minute).Result()

	if err != nil {
		return nil, err
	}

	if !res {
		return nil, ErrFailedToPreemptLock
	}

	return NewLock(c.client, key, value, expiration), nil
}

// Unlock都是有lock才会做的事了
func (l *Lock) Unlock(ctx context.Context, key string) error {
	//use lua,get and check ,then del
	res, err := l.client.Eval(ctx, luaUnlock, []string{l.key}, l.value).Int64()
	// 会出现什么情况，获取不到锁，因为那把锁不是你的了
	defer func() {
		l.unlock <- struct{}{}
		close(l.unlock)
	}()
	// 当锁不是你的
	if err == redis.Nil {
		fmt.Println("the integer return when err is redis.Nil")
		return ErrLockNotHold
	}
	if err != nil {
		return err
	}

	if res != 1 {
		return ErrLockNotHold
	}
	return nil
}
