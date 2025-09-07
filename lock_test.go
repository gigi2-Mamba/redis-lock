package redis_lock

import (
	"context"
	"errors"
	"github.com/golang/mock/gomock"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"redis-clock/mocks"
	"testing"
	"time"
)

func TestClient_TryLock(t *testing.T) {
	// go mock怎么需要一个这样的controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testCases := []struct {
		name string

		// mock something need to be mocked
		mock func() redis.Cmdable
		// input
		key        string
		expiration time.Duration

		//output
		wantLock *Lock
		wantErr  error
	}{
		// TODO: Add test cases.
		{
			name:       "locked", // locked successful  case
			key:        "locked-key",
			expiration: time.Minute,
			mock: func() redis.Cmdable {
				rdb := mocks.NewMockCmdable(ctrl) // 依赖mock都会帮你mock,当你需要看mock 那么回到生成的目录包，mocks
				// mocks.NewMockTarget
				res := redis.NewBoolResult(true, nil)
				rdb.EXPECT(). //
					SetNX(gomock.Any(), "locked-key", gomock.Any(), time.Minute).
					Return(res)
				return rdb
			},
			wantLock: &Lock{
				key:        "locked-key",
				expiration: time.Minute,
			},
		},
		{
			name:       "network error", // locked successful  case
			key:        "network-key",
			expiration: time.Minute,
			mock: func() redis.Cmdable {
				rdb := mocks.NewMockCmdable(ctrl) // 依赖mock都会帮你mock,当你需要看mock 那么回到生成的目录包，mocks
				// mocks.NewMockTarget
				res := redis.NewBoolResult(false, errors.New("network failed"))
				rdb.EXPECT(). //
					SetNX(gomock.Any(), "network-key", gomock.Any(), time.Minute).
					Return(res)
				return rdb
			},
			wantErr: errors.New("network failed"),
		},
		{
			name:       "locked fail", // locked successful  case
			key:        "locked-fail",
			expiration: time.Minute,
			mock: func() redis.Cmdable {
				rdb := mocks.NewMockCmdable(ctrl) // 依赖mock都会帮你mock,当你需要看mock 那么回到生成的目录包，mocks
				// mocks.NewMockTarget
				res := redis.NewBoolResult(false, nil)
				rdb.EXPECT(). //
					SetNX(gomock.Any(), "locked-fail", gomock.Any(), time.Minute).
					Return(res)
				return rdb
			},
			wantErr: ErrFailedToPreemptLock,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//var c Client // 必须的起点
			c := NewClient(tc.mock())

			l, err := c.TryLock(context.Background(), tc.key, tc.expiration)
			//fmt.Println("err after trylock : ", err.Error())
			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}
			assert.NotNil(t, l.client) // 感觉有点刻意
			assert.Equal(t, tc.wantLock.key, l.key)
			assert.NotEmpty(t, l.value)
		})
	}
}
