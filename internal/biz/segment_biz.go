package biz

import (
	"context"
	"fmt"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"go.uber.org/atomic"
	v1 "seg-server/api/segment/v1"
	"seg-server/internal/biz/model"
	"sync"
	"time"
)

var (
	// ErrUserNotFound is user not found.
	ErrUserNotFound = errors.NotFound(v1.ErrorReason_USER_NOT_FOUND.String(), "user not found")
	ErrDBOps        = errors.NotFound(v1.ErrorReason_USER_NOT_FOUND.String(), "update and get id error")
	// ErrTagNotFound key不存在时的异常码
	ErrTagNotFound = errors.NotFound(v1.ErrorReason_USER_NOT_FOUND.String(), "biz tag not found")
)

// Segment is a Segment model.
type Segment struct {
	Hello string
}

// SegmentRepo is a Greater repo.
type SegmentRepo interface {
	UpdateAndGetMaxId(ctx context.Context, tag string) (seg model.LeafAlloc, err error)
	GetLeafAlloc(ctx context.Context, tag string) (seg model.LeafAlloc, err error)
	GetAllTags(ctx context.Context) (tags []string, err error)
	GetAllLeafAllocs(ctx context.Context) (leafs []*model.LeafAlloc, err error)
}

// SegmentUsecase is a Segment usecase.
type SegmentUsecase struct {
	repo SegmentRepo
	segs map[string]*model.Segment
	bl   sync.RWMutex

	log *log.Helper
}

// NewSegmentUsecase new a Segment usecase.
func NewSegmentUsecase(repo SegmentRepo, logger log.Logger) *SegmentUsecase {
	s := &SegmentUsecase{
		repo: repo,
		segs: make(map[string]*model.Segment),
		log:  log.NewHelper(log.With(logger, "module", "segment/biz"))}

	s.loadSeqs()
	go s.loadProc()

	return s
}

func (uc *SegmentUsecase) nextStep(ctx context.Context, b *model.Segment) (err error) {

	leafAlloc, err := uc.repo.UpdateAndGetMaxId(ctx, b.GetKey())
	if err != nil {
		uc.log.Error("nextStep, db error : ", err)
		return fmt.Errorf("db error : %s %w", err, ErrDBOps)
	}

	value := leafAlloc.MaxId - int64(leafAlloc.Step)
	b.GetValue().Store(value)
	b.MaxId = leafAlloc.MaxId
	b.Step = leafAlloc.Step

	return
}

// GetID creates a Segment, and returns the new Segment.
func (uc *SegmentUsecase) GetID(ctx context.Context, tag string) (int64, error) {

	uc.bl.RLock()
	seg, ok := uc.segs[tag]
	uc.bl.RUnlock()

	if !ok {
		return 0, ErrTagNotFound
	}

	seg.Lock()
	// NOTE: make sure curSeq begin with maxSeq when start from 0
	if seg.GetValue().Load() == 0 || seg.GetValue().Load()+1 > seg.MaxId {
		if err := uc.nextStep(ctx, seg); err != nil {
			seg.Unlock()
			return 0, ErrDBOps
		}
	}
	seg.Unlock()

	idRet := seg.GetValue().Load()
	seg.GetValue().Inc()

	return idRet, nil
}

// loadProc 定时1min同步一次db和cache
func (uc *SegmentUsecase) loadProc() {
	ticker := time.NewTicker(time.Minute)

	for {
		select {
		case <-ticker.C:
			uc.loadSeqs()
		}
	}
}

func (uc *SegmentUsecase) loadSeqs() (err error) {

	leafAllocs, err := uc.repo.GetAllLeafAllocs(context.TODO())
	if err != nil {
		return err
	}

	for _, leafAlloc := range leafAllocs {
		uc.bl.Lock()
		if _, ok := uc.segs[leafAlloc.BizTag]; !ok {
			uc.segs[leafAlloc.BizTag] = &model.Segment{
				BizKey: leafAlloc.BizTag,
				Value:  atomic.NewInt64(0),
				MaxId:  leafAlloc.MaxId,
				Step:   leafAlloc.Step,
			}
			uc.bl.Unlock()
			continue
		}
		uc.bl.Unlock()
	}

	return nil
}
