package data

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	clientv3 "go.etcd.io/etcd/client/v3"
	"seg-server/internal/biz"
)

type SnowflakeRepoIml struct {
	data *Data
	log  *log.Helper
}

func (s *SnowflakeRepoIml) GetPrefixKey(ctx context.Context, prefix string) (*clientv3.GetResponse, error) {

	return s.data.etcdCli.Get(ctx, prefix, clientv3.WithPrefix())
}

// CreateKeyWithOptLock 事务乐观锁创建
func (s *SnowflakeRepoIml) CreateKeyWithOptLock(ctx context.Context, key string, val string) bool {

	txnResponse, err := s.data.etcdCli.Txn(ctx).
		If(clientv3.Compare(clientv3.CreateRevision(key), "=", 0)).
		Then(clientv3.OpPut(key, val)).Commit()
	if err != nil {
		return false
	}
	return txnResponse.Succeeded
}

func (s *SnowflakeRepoIml) CreateOrUpdateKey(ctx context.Context, key string, val string) bool {

	_, err := s.data.etcdCli.Put(ctx, key, val)
	if err != nil {
		return false
	}

	return true
}

func (s *SnowflakeRepoIml) GetKey(ctx context.Context, key string) (*clientv3.GetResponse, error) {

	return s.data.etcdCli.Get(ctx, key)
}

// NewSnowflakeRepo .
func NewSnowflakeRepo(data *Data, logger log.Logger) biz.SnowflakeIDGenRepo {
	return &SnowflakeRepoIml{
		data: data,
		log:  log.NewHelper(log.With(logger, "module", "leaf-grpc-repo/snowflake")),
	}
}
