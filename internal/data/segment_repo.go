package data

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"go.uber.org/atomic"
	"gorm.io/gorm"
	"seg-server/internal/biz"
	"seg-server/internal/data/model"
)

const (
	TableName    = "leaf_alloc" // 写到配置中
	_maxSeqSQL   = "SELECT max_id FROM biz_tag WHERE id=?"
	_upMaxSeqSQL = "UPDATE biz_tag SET max_id=? WHERE id=? AND max_id=?"
)

type SegmentIdRepo struct {
	data *Data
	model.Segment

	log *log.Helper
}

func (s *SegmentIdRepo) GetAllLeafAllocs(ctx context.Context) (leafs []*biz.LeafAlloc, err error) {
	if err = s.data.db.WithContext(ctx).Select("biz_tag",
		"max_id", "step", "update_at").Find(&leafs).Error; err != nil {

		return nil, err
	}

	return
}

func (s *SegmentIdRepo) GetLeafAlloc(ctx context.Context, tag string) (seg biz.LeafAlloc, err error) {
	if err = s.data.db.WithContext(ctx).Select("biz_tag",
		"max_id", "step").First(&seg).Error; err != nil {

		return
	}

	return
}

func (s *SegmentIdRepo) GetAllTags(ctx context.Context) (tags []string, err error) {
	if err = s.data.db.WithContext(ctx).
		Pluck("biz_tag", &tags).Error; err != nil {

		return
	}

	return
}

func (s *SegmentIdRepo) UpdateAndGetMaxId(ctx context.Context, tag string) (seg biz.LeafAlloc, err error) {

	// Begin
	// UPDATE table SET max_id=max_id+step WHERE biz_tag=xxx
	// SELECT tag, max_id, step FROM table WHERE biz_tag=xxx
	// Commit
	err = s.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err = tx.Table(s.data.tableName).Where("biz_tag =?", tag).
			Update("max_id", "max_id + step").Error; err != nil {

			return err
		}

		if err = tx.Table(s.data.tableName).Select("biz_tag",
			"max_id", "step").Find(&seg).Error; err != nil {

			return err
		}

		return nil
	})

	return
}

func (s *SegmentIdRepo) GetValue() *atomic.Int64 {
	return s.Value
}

func (s *SegmentIdRepo) SetValue(value *atomic.Int64) {
	s.Value = value
}

func (s *SegmentIdRepo) GetMax() int64 {
	return s.MaxId
}

func (s *SegmentIdRepo) SetMax(max int64) {
	s.MaxId = max
}

func (s *SegmentIdRepo) GetStep() int {
	return s.Step
}

func (s *SegmentIdRepo) SetStep(step int) {
	s.Step = step
}

func (s *SegmentIdRepo) GetIdle() int64 {
	value := s.GetValue().Load()
	return s.GetMax() - value
}

// NewSegmentIDRepo .
func NewSegmentIDRepo(data *Data, logger log.Logger) biz.SegmentRepo {
	return &SegmentIdRepo{
		data:    data,
		Segment: model.Segment{Value: atomic.NewInt64(0)},
		log:     log.NewHelper(log.With(logger, "module", "segment-repo/data")),
	}
}
