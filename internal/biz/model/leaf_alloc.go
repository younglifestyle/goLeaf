package model

import "time"

type LeafAlloc struct {
	//ID          uint   `gorm:"primaryKey" json:"-"`
	BizTag      string    `xorm:"varchar2(64) PRIMARY KEY unique 'biz_tag'" json:"biz_tag"`
	MaxId       int64     `xorm:"number 'max_id'" json:"max_id"`
	Step        int       `xorm:"number 'step'" json:"step"`
	Description string    `xorm:"varchar2(512) 'description'" json:"description"`
	UpdatedAt   time.Time `xorm:"TIMESTAMP(6) WITH LOCAL TIME ZONE default CURRENT_TIMESTAMP 'update_time'" json:"update_time,omitempty"`
	CreatedAt   time.Time `xorm:"TIMESTAMP(6) WITH LOCAL TIME ZONE default CURRENT_TIMESTAMP 'created_time'" json:"created_time,omitempty"`
}
