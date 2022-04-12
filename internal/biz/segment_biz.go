package biz

import (
	"context"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"go.uber.org/atomic"
	v1 "seg-server/api/helloworld/v1"
	"sync"
)

var (
	// ErrUserNotFound is user not found.
	ErrUserNotFound = errors.NotFound(v1.ErrorReason_USER_NOT_FOUND.String(), "user not found")
)

// Segment is a Segment model.
type Segment struct {
	Hello string
}

type LeafAlloc struct {
	//ID          uint   `gorm:"primaryKey" json:"-"`
	BizTag      string `gorm:"column:biz_tag; type:VARCHAR(128) not null; primaryKey" json:"biz_tag"`
	MaxId       int64  `gorm:"column:max_id; type:BIGINT(20) not null default 1" json:"max_id"`
	Step        int    `gorm:"column:step; type:INT(11) not null default 0" json:"step"`
	Description string `gorm:"column:description; type:VARCHAR(256) not null" json:"description"`
	UpdatedAt   int64  `gorm:"autoUpdateTime:milli;column:updated_time" json:"updated_time,omitempty"`
	CreatedAt   int64  `gorm:"autoCreateTime:milli;column:created_time" json:"created_time,omitempty"`
}

// SegmentRepo is a Greater repo.
type SegmentRepo interface {
	UpdateAndGetMaxId(ctx context.Context, tag string) (seg LeafAlloc, err error)
	GetLeafAlloc(ctx context.Context, tag string) (seg LeafAlloc, err error)
	GetAllTags(ctx context.Context) (tags []string, err error)
	GetAllLeafAllocs(ctx context.Context) (leafs []*LeafAlloc, err error)
	GetValue() *atomic.Int64
	SetValue(value *atomic.Int64)
	GetMax() int64
	SetMax(max int64)
	GetStep() int
	SetStep(step int)
	GetIdle() int64
}

// SegmentUsecase is a Segment usecase.
type SegmentUsecase struct {
	repo  SegmentRepo
	leafs map[string]*LeafAlloc

	log *log.Helper
	sync.RWMutex
}

// NewSegmentUsecase new a Segment usecase.
func NewSegmentUsecase(repo SegmentRepo, logger log.Logger) *SegmentUsecase {
	return &SegmentUsecase{
		repo:  repo,
		leafs: make(map[string]*LeafAlloc),
		log:   log.NewHelper(log.With(logger, "module", "segment/biz"))}
}

// GetID creates a Segment, and returns the new Segment.
func (uc *SegmentUsecase) GetID(ctx context.Context, g *Segment) (*Segment, error) {
	uc.log.WithContext(ctx).Infof("GetID: %v", g.Hello)
	return &Segment{}, nil
}
