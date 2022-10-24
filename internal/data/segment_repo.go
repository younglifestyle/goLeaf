package data

import (
	"context"
	"errors"
	"github.com/go-kratos/kratos/v2/log"
	"seg-server/internal/biz"
	"seg-server/internal/biz/model"
	"xorm.io/xorm"
)

type SegmentIdGenRepoIml struct {
	data *Data
	log  *log.Helper
}

func (s *SegmentIdGenRepoIml) GetAllLeafAllocs(ctx context.Context) (leafs []*model.LeafAlloc, err error) {
	err = s.data.db.Table(s.data.tableName).Context(ctx).Find(&leafs)
	if err != nil {
		return nil, err
	}

	return
}

func (s *SegmentIdGenRepoIml) GetLeafAlloc(ctx context.Context, tag string) (seg model.LeafAlloc, err error) {

	_, err = s.data.db.Table(s.data.tableName).Context(ctx).Select("biz_tag, max_id, step").
		Where("biz_tag = ?", tag).Get(&seg)
	if err != nil {
		return
	}

	return
}

func (s *SegmentIdGenRepoIml) GetAllTags(ctx context.Context) (tags []string, err error) {
	err = s.data.db.Table(s.data.tableName).Context(ctx).Cols("biz_tag").Find(&tags)
	if err != nil {
		return nil, err
	}

	return
}

func (s *SegmentIdGenRepoIml) UpdateAndGetMaxId(ctx context.Context, tag string) (leafAlloc model.LeafAlloc, err error) {

	_, err = s.data.db.Transaction(func(tx *xorm.Session) (interface{}, error) {
		if _, err := tx.Table(s.data.tableName).Context(ctx).
			SetExpr("max_id", `"max_id" + "step"`).
			Update(&model.LeafAlloc{}, &model.LeafAlloc{
				BizTag: tag,
			}); err != nil {

			return nil, err
		}

		// as conditions
		leafAlloc.BizTag = tag
		has, err := tx.Table(s.data.tableName).Context(ctx).Select(`"biz_tag", "max_id", "step"`).
			Get(&leafAlloc)
		if err != nil {
			return nil, err
		}
		if !has {
			return nil, errors.New("no record")
		}

		return nil, nil
	})

	return
}

func (s *SegmentIdGenRepoIml) UpdateMaxIdByCustomStepAndGetLeafAlloc(ctx context.Context, tag string, step int) (leafAlloc model.LeafAlloc, err error) {

	_, err = s.data.db.Transaction(func(tx *xorm.Session) (interface{}, error) {
		if _, err := tx.Table(s.data.tableName).Context(ctx).
			Incr("max_id", step).
			Update(&model.LeafAlloc{}, &model.LeafAlloc{
				BizTag: tag,
			}); err != nil {

			return nil, err
		}

		// as conditions
		leafAlloc.BizTag = tag
		has, err := tx.Table(s.data.tableName).Context(ctx).Select(`"biz_tag", "max_id", "step"`).
			Get(&leafAlloc)
		if err != nil {
			return nil, err
		}
		if !has {
			return nil, errors.New("no record")
		}
		return nil, nil
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
