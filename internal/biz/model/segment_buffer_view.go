package model

type SegmentBufferView struct {
	Key       string
	Value0    int64
	Step0     int
	Max0      int64
	Value1    int64
	Step1     int
	Max1      int64
	Pos       int
	NextReady bool
	InitOk    bool
	AutoClean bool
}
