package data

import (
	"context"
	"gorm.io/gorm"
	"seg-server/internal/model"
)

const (
	TableName    = "leaf_alloc" // 写到配置中
	_maxSeqSQL   = "SELECT max_id FROM biz_tag WHERE id=?"
	_upMaxSeqSQL = "UPDATE biz_tag SET max_id=? WHERE id=? AND max_id=?"
)

func (d *Data) UpdateAndGetMaxId(ctx context.Context, tag string) (seg model.LeafAlloc, err error) {

	// Begin
	// UPDATE table SET max_id=max_id+step WHERE biz_tag=xxx
	// SELECT tag, max_id, step FROM table WHERE biz_tag=xxx
	// Commit
	err = d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err = tx.Table(TableName).Where("biz_tag =?", tag).
			Update("max_id", "max_id + step").Error; err != nil {

			return err
		}

		if err = tx.Table(TableName).Select("biz_tag",
			"max_id", "step").Find(&seg).Error; err != nil {

			return err
		}

		return nil
	})

	return
}
