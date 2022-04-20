package model

import "time"

type LeafAlloc struct {
	//ID          uint   `gorm:"primaryKey" json:"-"`
	BizTag      string    `gorm:"column:biz_tag; type:VARCHAR(128) not null; primaryKey" json:"biz_tag"`
	MaxId       int64     `gorm:"column:max_id; type:BIGINT(20) not null default 1" json:"max_id"`
	Step        int       `gorm:"column:step; type:INT(11) not null default 0" json:"step"`
	Description string    `gorm:"column:description; type:VARCHAR(256) not null" json:"description"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime:milli;column:update_time" json:"update_time,omitempty"`
	CreatedAt   time.Time `gorm:"autoCreateTime:milli;column:created_time" json:"created_time,omitempty"`
}
