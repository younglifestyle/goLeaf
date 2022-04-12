package model

import "go.uber.org/atomic"

type Segment struct {
	Value *atomic.Int64
	MaxId int64
	Step  int
}
