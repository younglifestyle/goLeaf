package data

import (
	"context"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"goLeaf/internal/biz"
	"goLeaf/internal/biz/model"
	"gorm.io/gorm"
	"strings"
)

type SegmentIdGenRepoIml struct {
	data *Data
	log  *log.Helper
}

func (s *SegmentIdGenRepoIml) CleanLeafMaxId(ctx context.Context, tags []string) (err error) {
	err = s.data.db.Table(s.data.tableName).WithContext(ctx).
		Where("biz_tag IN ?", tags).Update("max_id", 0).Error
	if err != nil {
		return err
	}

	return nil
}

func (s *SegmentIdGenRepoIml) SaveLeafAlloc(ctx context.Context, leafAlloc *model.LeafAlloc) error {

	err := s.data.db.Table(s.data.tableName).WithContext(ctx).Create(leafAlloc).Error
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate") {
			return fmt.Errorf("%s, %s, %w", "save leaf info error", err, biz.ErrIDExist)
		}

		return fmt.Errorf("%s, %s, %w", "save leaf info error", err, biz.ErrDBOps)
	}

	return nil
}

func (s *SegmentIdGenRepoIml) GetAllLeafAllocs(ctx context.Context) (leafs []*model.LeafAlloc, err error) {
	if err = s.data.db.Table(s.data.tableName).
		WithContext(ctx).Find(&leafs).Error; err != nil {

		return nil, err
	}

	return
}

func (s *SegmentIdGenRepoIml) GetLeafAlloc(ctx context.Context, tag string) (seg model.LeafAlloc, err error) {
	if err = s.data.db.Table(s.data.tableName).WithContext(ctx).Select("biz_tag",
		"max_id", "step").Where("biz_tag = ?", tag).First(&seg).Error; err != nil {
		return
	}

	return
}

func (s *SegmentIdGenRepoIml) GetAllTags(ctx context.Context) (tags []string, err error) {
	if err = s.data.db.Table(s.data.tableName).WithContext(ctx).
		Pluck("biz_tag", &tags).Error; err != nil {

		return
	}

	return
}

func (s *SegmentIdGenRepoIml) UpdateAndGetMaxId(ctx context.Context, tag string) (leafAlloc model.LeafAlloc, err error) {

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
			"max_id", "step", "auto_clean").
			Where("biz_tag = ?", tag).First(&leafAlloc).Error; err != nil {

			return err
		}

		return nil
	})

	return
}

func (s *SegmentIdGenRepoIml) UpdateMaxIdByCustomStepAndGetLeafAlloc(ctx context.Context, tag string, step int) (leafAlloc model.LeafAlloc, err error) {

	err = s.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err = tx.Table(s.data.tableName).Where("biz_tag = ?", tag).
			Update("max_id", gorm.Expr("max_id + ?", step)).Error; err != nil {

			return err
		}

		if err = tx.Table(s.data.tableName).Select("biz_tag",
			"max_id", "step", "auto_clean").Where("biz_tag = ?", tag).First(&leafAlloc).Error; err != nil {

			return err
		}

		return nil
	})

	return
}

// NewSegmentIdGenRepo .
func NewSegmentIdGenRepo(data *Data, logger log.Logger) biz.SegmentIDGenRepo {
	return &SegmentIdGenRepoIml{
		data: data,
		log:  log.NewHelper(log.With(logger, "module", "leaf-grpc-repo/segment")),
	}
}
