package model

import (
	"go.uber.org/atomic"
	"sync"
)

type Segment struct {
	BizKey string
	Value  *atomic.Int64
	MaxId  int64
	Step   int

	sync.Mutex
}

func (s *Segment) GetKey() string {
	return s.BizKey
}

func (s *Segment) GetValue() *atomic.Int64 {
	return s.Value
}

func (s *Segment) SetValue(value *atomic.Int64) {
	s.Value = value
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
	value := s.GetValue().Load()
	return s.GetMax() - value
}
