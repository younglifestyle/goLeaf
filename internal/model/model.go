package model

import "sync"

type LeafAlloc struct {
	//ID          uint   `gorm:"primaryKey" json:"-"`
	BizTag      string `gorm:"column:biz_tag; type:VARCHAR(128) not null; primaryKey" json:"biz_tag"`
	MaxId       int64  `gorm:"column:max_id; type:BIGINT(20) not null default 1" json:"max_id"`
	Step        int    `gorm:"column:step; type:INT(11) not null default 0" json:"step"`
	Description string `gorm:"column:description; type:VARCHAR(256) not null" json:"description"`
	CreatedAt   int64  `gorm:"autoCreateTime:milli"`
	UpdatedAt   int64  `gorm:"autoUpdateTime:milli"`
	DeletedAt   int64
	Updated     string     `gorm:"-" json:"updated"`
	Mutex       sync.Mutex `gorm:"-" json:"-"`
}
