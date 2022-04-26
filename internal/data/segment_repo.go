package data

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	clientv3 "go.etcd.io/etcd/client/v3"
	"gorm.io/gorm"
	"seg-server/internal/biz"
	"seg-server/internal/biz/model"
)

type SegmentIdRepo struct {
	data *Data
	log  *log.Helper
}

func (s *SegmentIdRepo) GetAllLeafAllocs(ctx context.Context) (leafs []*model.LeafAlloc, err error) {
	if err = s.data.db.Table(s.data.tableName).
		WithContext(ctx).Find(&leafs).Error; err != nil {

		return nil, err
	}

	return
}

func (s *SegmentIdRepo) GetLeafAlloc(ctx context.Context, tag string) (seg model.LeafAlloc, err error) {
	if err = s.data.db.Table(s.data.tableName).WithContext(ctx).Select("biz_tag",
		"max_id", "step").Where("biz_tag = ?", tag).First(&seg).Error; err != nil {

		return
	}

	return
}

func (s *SegmentIdRepo) GetAllTags(ctx context.Context) (tags []string, err error) {
	if err = s.data.db.Table(s.data.tableName).WithContext(ctx).
		Pluck("biz_tag", &tags).Error; err != nil {

		return
	}

	return
}

func (s *SegmentIdRepo) UpdateAndGetMaxId(ctx context.Context, tag string) (leafAlloc model.LeafAlloc, err error) {

	// Begin
	// UPDATE table SET max_id=max_id+step WHERE biz_tag=xxx
	// SELECT tag, max_id, step FROM table WHERE biz_tag=xxx
	// Commit
	err = s.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err = tx.Table(s.data.tableName).Where("biz_tag =?", tag).
			Update("max_id", gorm.Expr("max_id + step")).Error; err != nil {

			return err
		}

		if err = tx.Table(s.data.tableName).Select("biz_tag",
			"max_id", "step").Where("biz_tag = ?", tag).First(&leafAlloc).Error; err != nil {

			return err
		}

		return nil
	})

	return
}

func (s *SegmentIdRepo) UpdateMaxIdByCustomStepAndGetLeafAlloc(ctx context.Context, tag string, step int) (leafAlloc model.LeafAlloc, err error) {

	err = s.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err = tx.Table(s.data.tableName).Where("biz_tag = ?", tag).
			Update("max_id", gorm.Expr("max_id + ?", step)).Error; err != nil {

			return err
		}

		if err = tx.Table(s.data.tableName).Select("biz_tag",
			"max_id", "step").Where("biz_tag = ?", tag).First(&leafAlloc).Error; err != nil {

			return err
		}

		return nil
	})

	return
}

func (s *SegmentIdRepo) GetPrefixKey(ctx context.Context, prefix string) (*clientv3.GetResponse, error) {

	return s.data.etcdCli.Get(ctx, prefix, clientv3.WithPrefix())
}

// CreateKey 事务乐观锁创建
func (s *SegmentIdRepo) CreateKeyWithOptLock(ctx context.Context, key string, val string) bool {

	txnResponse, err := s.data.etcdCli.Txn(ctx).
		If(clientv3.Compare(clientv3.CreateRevision(key), "=", 0)).
		Then(clientv3.OpPut(key, val)).Commit()
	if err != nil {
		return false
	}
	return txnResponse.Succeeded
}

func (s *SegmentIdRepo) CreateOrUpdateKey(ctx context.Context, key string, val string) bool {

	_, err := s.data.etcdCli.Put(ctx, key, val)
	if err != nil {
		return false
	}

	return true
}

func (s *SegmentIdRepo) GetKey(ctx context.Context, key string) (*clientv3.GetResponse, error) {

	return s.data.etcdCli.Get(ctx, key)
}

// NewSegmentIDRepo .
func NewSegmentIDRepo(data *Data, logger log.Logger) biz.SegmentRepo {
	return &SegmentIdRepo{
		data: data,
		log:  log.NewHelper(log.With(logger, "module", "leaf-grpc-repo/data")),
	}
}
