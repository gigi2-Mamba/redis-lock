package redis_lock

import (
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// test function 都是用的标准库的test包
func TestClient_Try_lock(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", //only addr，差不多是这样
		Password: "",
		DB:       0,
	})
	ping := rdb.Ping(context.Background())
	if ping.Err() != nil {
		ping.Err()
	}
	testCases := []struct {
		name string
		//prepare data
		before func()
		//verify & clear
		after func()
		//input
		key        string
		expiration time.Duration
		//output
		wantLock *Lock
		wantErr  error
	}{
		{
			name:   "locked",
			before: func() {}, // 加锁在调用前不需要准备什么数据
			after: func() {
				res, err := rdb.Del(context.Background(), "locked").Result()
				require.NoError(t, err)
				require.Equal(t, int64(1), res)
			},
			key:        "locked",
			expiration: time.Minute,
		},
	}

	for _, tc := range testCases { // 对test cases 循环处理这很正常。
		t.Run(tc.name, func(t *testing.T) { // 这些模板都是必备的
			tc.before() // 很明显就是两个钩子
			c := NewClient(rdb)
			l, err := c.TryLock(context.Background(), tc.key, tc.expiration)
			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}
			assert.NotNil(t, l.client, nil)
			assert.Equal(t, tc.key, l.key)
			assert.NotEmpty(t, l.value)
			tc.after()
		})
	}

}
