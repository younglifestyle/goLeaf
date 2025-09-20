package biz

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	clientv3 "go.etcd.io/etcd/client/v3"
	"goLeaf/internal/conf"
)

type noopSnowflakeRepo struct{}

func (noopSnowflakeRepo) GetPrefixKey(ctx context.Context, prefix string) (*clientv3.GetResponse, error) {
	return nil, nil
}

func (noopSnowflakeRepo) CreateKeyWithOptLock(ctx context.Context, key string, val string) bool {
	return true
}

func (noopSnowflakeRepo) CreateOrUpdateKey(ctx context.Context, key string, val string) bool {
	return true
}

func (noopSnowflakeRepo) GetKey(ctx context.Context, key string) (*clientv3.GetResponse, error) {
	return nil, nil
}

func TestGetSnowflakeIDRejectsClockRollback(t *testing.T) {
	uc := &SnowflakeIdGenUsecase{
		repo: noopSnowflakeRepo{},
		conf: &conf.Bootstrap{
			Data: &conf.Data{
				Etcd: &conf.Data_Etcd{Endpoints: []string{"127.0.0.1:12379"}, SnowflakeEnable: true, TimeDeviation: 10},
			},
		},
		twepoch:  twepoch,
		workerId: 1,
		log:      log.NewHelper(log.NewStdLogger(io.Discard)),
	}

	uc.lastTimestamp = time.Now().Add(200 * time.Millisecond).UnixMilli()

	if _, err := uc.GetSnowflakeID(context.Background()); err == nil || !errors.Is(err, ErrSnowflakeTimeException) {
		t.Fatalf("expected clock rollback to trigger ErrSnowflakeTimeException, got %v", err)
	}
}
