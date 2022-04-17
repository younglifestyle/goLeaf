package model

import (
	"go.uber.org/atomic"
	"sync"
)

type SegmentBuffer struct {
	SegmentBufferParam
	Segments []*Segment
}

type SegmentBufferParam struct {
	BizKey        string
	RWMutex       *sync.RWMutex
	CurrentPos    int          // 当前的使用的segment的index
	NextReady     bool         // 下一个segment是否处于可切换状态
	InitOk        bool         // DB的数据段数据是否初始化
	ThreadRunning *atomic.Bool // 是否更新下一ID段的线程，是否在运行中
	Step          int
	UpdatedTime   int64
}

func NewSegmentBuffer() *SegmentBuffer {
	s := new(SegmentBuffer)
	s.Segments = make([]*Segment, 0, 0)
	segment1 := NewSegment(s)
	segment2 := NewSegment(s)
	s.Segments = append(s.Segments, segment1, segment2)
	s.CurrentPos = 0
	s.NextReady = false
	s.InitOk = false
	s.ThreadRunning = atomic.NewBool(false)
	s.RWMutex = &sync.RWMutex{}
	return s
}

func (segbf *SegmentBuffer) GetKey() string {
	return segbf.BizKey
}

func (segbf *SegmentBuffer) SetKey(key string) {
	segbf.BizKey = key
}

func (segbf *SegmentBuffer) GetSegments() []*Segment {
	return segbf.Segments
}

func (segbf *SegmentBuffer) GetAnotherSegment() *Segment {
	return segbf.Segments[segbf.NextPos()]
}

func (segbf *SegmentBuffer) GetCurrent() *Segment {
	return segbf.Segments[segbf.CurrentPos]
}

func (segbf *SegmentBuffer) GetCurrentPos() int {
	return segbf.CurrentPos
}

func (segbf *SegmentBuffer) NextPos() int {
	return (segbf.CurrentPos + 1) % 2
}

func (segbf *SegmentBuffer) SwitchPos() {
	segbf.CurrentPos = segbf.NextPos()
}

func (segbf *SegmentBuffer) IsInitOk() bool {
	return segbf.InitOk
}

func (segbf *SegmentBuffer) SetInitOk(initOk bool) {
	segbf.InitOk = initOk
}

func (segbf *SegmentBuffer) IsNextReady() bool {
	return segbf.NextReady
}

func (segbf *SegmentBuffer) SetNextReady(nextReady bool) {
	segbf.NextReady = nextReady
}

func (segbf *SegmentBuffer) GetThreadRunning() *atomic.Bool {
	return segbf.ThreadRunning
}

//func (segbf *SegmentBuffer) RLock() {
//	segbf.RWMutex.RLock()
//}
//
//func (segbf *SegmentBuffer) RUnLock() {
//	segbf.RWMutex.RUnlock()
//}
//
//func (segbf *SegmentBuffer) WLock() {
//	segbf.RWMutex.Lock()
//}
//
//func (segbf *SegmentBuffer) WUnLock() {
//	segbf.RWMutex.Unlock()
//}

func (segbf *SegmentBuffer) GetStep() int {
	return segbf.Step
}

func (segbf *SegmentBuffer) SetStep(step int) {
	segbf.Step = step
}

func (segbf *SegmentBuffer) GetUpdateTimeStamp() int64 {
	return segbf.UpdatedTime
}

func (segbf *SegmentBuffer) SetUpdateTimeStamp(ts int64) {
	segbf.UpdatedTime = ts
}
