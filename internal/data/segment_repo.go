package data

import (
	"context"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"seg-server/internal/biz"
	"seg-server/internal/biz/model"
	"time"
	"xorm.io/xorm"
)

var updateMaxIdFormStep = `UPDATE "%s" SET "max_id"="max_id" + "step", "update_time" = TO_TIMESTAMP('%s', 'YYYY-MM-DD HH24:MI:SS.FF6') WHERE "biz_tag"='%s'`
var updateMaxIdFormCustomStep = `UPDATE "%s" SET "max_id"="max_id" + %d, "update_time" = TO_TIMESTAMP('%s', 'YYYY-MM-DD HH24:MI:SS.FF6') WHERE "biz_tag"='%s'`
var saveLeafSql = `INSERT INTO "%s" ("biz_tag", "max_id", "step", "description", "update_time", "created_time") 
VALUES ('%s', %d, %d, '%s', TO_TIMESTAMP('%s', 'YYYY-MM-DD HH24:MI:SS.FF6'), TO_TIMESTAMP('%s', 'YYYY-MM-DD HH24:MI:SS.FF6'))`

type SegmentIdGenRepoIml struct {
	data *Data
	log  *log.Helper
}

func (s *SegmentIdGenRepoIml) SaveLeafAlloc(ctx context.Context, leafAlloc *model.LeafAlloc) error {

	_, err := s.data.db.Context(ctx).Exec(fmt.Sprintf(saveLeafSql,
		s.data.tableName,
		leafAlloc.BizTag, leafAlloc.MaxId, leafAlloc.Step, leafAlloc.Description,
		time.Unix(time.Now().Unix(), 0).Format("2006-01-02 15:04:05.0000"),
		time.Unix(time.Now().Unix(), 0).Format("2006-01-02 15:04:05.0000")))
	if err != nil {
		return fmt.Errorf("%s, %s, %w", "save leaf info error", err, biz.ErrDBOps)
	}

	return nil
}

func (s *SegmentIdGenRepoIml) GetAllLeafAllocs(ctx context.Context) (leafs []*model.LeafAlloc, err error) {
	err = s.data.db.Table(s.data.tableName).Context(ctx).Find(&leafs)
	if err != nil {
		return nil, fmt.Errorf("%s, %s, %w", "find all leaf info error", err, biz.ErrDBOps)
	}

	return
}

func (s *SegmentIdGenRepoIml) GetLeafAlloc(ctx context.Context, tag string) (seg model.LeafAlloc, err error) {

	_, err = s.data.db.Table(s.data.tableName).Context(ctx).Select("biz_tag, max_id, step").
		Where(fmt.Sprintf(`"biz_tag" = '%s'`, tag)).Get(&seg)
	if err != nil {
		return
	}

	return
}

func (s *SegmentIdGenRepoIml) GetAllTags(ctx context.Context) (tags []string, err error) {
	err = s.data.db.Table(s.data.tableName).Context(ctx).Cols("biz_tag").Find(&tags)
	if err != nil {
		return nil, fmt.Errorf("%s, %s, %w", "find all leaf tag error", err, biz.ErrDBOps)
	}

	return
}

func (s *SegmentIdGenRepoIml) UpdateAndGetMaxId(ctx context.Context, tag string) (leafAlloc model.LeafAlloc, err error) {

	_, err = s.data.db.Transaction(func(tx *xorm.Session) (interface{}, error) {

		res, err := tx.Context(ctx).Exec(fmt.Sprintf(updateMaxIdFormStep,
			s.data.tableName,
			time.Unix(time.Now().Unix(), 0).Format("2006-01-02 15:04:05.0000"), tag))
		if err != nil {
			return nil, fmt.Errorf("%s, %s, %w", "update leaf tag error", err, biz.ErrDBOps)
		}
		affected, err := res.RowsAffected()
		if affected == 0 {
			return nil, fmt.Errorf("%s, %w", "update leaf affected row is 0", biz.ErrDBOps)
		}

		// as conditions
		leafAlloc.BizTag = tag
		has, err := tx.Table(s.data.tableName).Context(ctx).Select(`"biz_tag", "max_id", "step"`).
			Get(&leafAlloc)
		if err != nil {
			return nil, fmt.Errorf("%s, %s, %w", "find leaf tag error", err, biz.ErrDBOps)
		}
		if !has {
			return nil, fmt.Errorf("%s, %s, %w", "find leaf tag error", "no record", biz.ErrDBOps)
		}

		return nil, nil
	})

	return
}

func (s *SegmentIdGenRepoIml) UpdateMaxIdByCustomStepAndGetLeafAlloc(ctx context.Context, tag string, step int) (leafAlloc model.LeafAlloc, err error) {

	_, err = s.data.db.Transaction(func(tx *xorm.Session) (interface{}, error) {
		res, err := tx.Context(ctx).Exec(fmt.Sprintf(updateMaxIdFormCustomStep,
			s.data.tableName,
			step,
			time.Unix(time.Now().Unix(), 0).Format("2006-01-02 15:04:05.0000"),
			tag))
		if err != nil {
			return nil, fmt.Errorf("%s, %s, %w", "update leaf tag error", err, biz.ErrDBOps)
		}
		affected, err := res.RowsAffected()
		if affected == 0 {
			return nil, fmt.Errorf("%s, %w", "update leaf affected row is 0", biz.ErrDBOps)
		}

		// as conditions
		leafAlloc.BizTag = tag
		has, err := tx.Table(s.data.tableName).Context(ctx).Select(`"biz_tag", "max_id", "step"`).
			Get(&leafAlloc)
		if err != nil {
			return nil, fmt.Errorf("%s, %s, %w", "find leaf tag error", err, biz.ErrDBOps)
		}
		if !has {
			return nil, fmt.Errorf("%s, %s, %w", "find leaf tag error", "no record", biz.ErrDBOps)
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
