package model

import (
	"go.uber.org/atomic"
)

type Segment struct {
	SegmentParam
	Buffer *SegmentBuffer // 当前号段所属的SegmentBuffer
}

type SegmentParam struct {
	Value *atomic.Int64 // 内存生成的每一个id号
	MaxId int64         // 当前号段允许的最大id值
	Step  int           // 步长 暂固定
}

func NewSegment(segmentBuffer *SegmentBuffer) *Segment {
	s := &Segment{
		SegmentParam: SegmentParam{
			Value: atomic.NewInt64(0),
		},
		Buffer: segmentBuffer,
	}
	return s
}

func (s *Segment) GetAndInc() int64 {
	return s.Value.Inc() - 1
}

func (s *Segment) GetValue() int64 {
	return s.Value.Load()
}

func (s *Segment) SetValue(value int64) {
	s.Value.Store(value)
}

func (s *Segment) GetMax() int64 {
	return s.MaxId
}

func (s *Segment) SetMax(max int64) {
	s.MaxId = max
}

func (s *Segment) GetStep() int {
	return s.Step
}

func (s *Segment) SetStep(step int) {
	s.Step = step
}

func (s *Segment) GetIdle() int64 {
	return s.GetMax() - s.Value.Load()
}

// GetBuffer 获取当前号段所属的SegmentBuffer
func (s *Segment) GetBuffer() *SegmentBuffer {
	return s.Buffer
}
