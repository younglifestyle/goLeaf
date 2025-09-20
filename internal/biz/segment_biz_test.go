package biz

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"goLeaf/internal/biz/model"
	"goLeaf/internal/conf"
)

type segmentRepoStub struct {
	baseStep       int
	lastCustomStep int
	customCalls    int
	getAllTags     func(ctx context.Context) ([]string, error)
}

func (s *segmentRepoStub) SaveLeafAlloc(ctx context.Context, leafAlloc *model.LeafAlloc) error {
	return nil
}

func (s *segmentRepoStub) UpdateAndGetMaxId(ctx context.Context, tag string) (model.LeafAlloc, error) {
	return model.LeafAlloc{BizTag: tag, Step: s.baseStep, MaxId: 1024, AutoClean: false}, nil
}

func (s *segmentRepoStub) GetLeafAlloc(ctx context.Context, tag string) (model.LeafAlloc, error) {
	return model.LeafAlloc{}, nil
}

func (s *segmentRepoStub) GetAllTags(ctx context.Context) ([]string, error) {
	if s.getAllTags != nil {
		return s.getAllTags(ctx)
	}
	return nil, nil
}

func (s *segmentRepoStub) GetAllLeafAllocs(ctx context.Context, tag string) ([]*model.LeafAlloc, error) {
	return nil, nil
}

func (s *segmentRepoStub) UpdateMaxIdByCustomStepAndGetLeafAlloc(ctx context.Context, tag string, step int) (model.LeafAlloc, error) {
	s.customCalls++
	s.lastCustomStep = step
	return model.LeafAlloc{BizTag: tag, Step: s.baseStep, MaxId: int64(step) + 1024, AutoClean: false}, nil
}

func (s *segmentRepoStub) CleanLeafMaxId(ctx context.Context, tags []string) error {
	return nil
}

func newTestSegmentUsecase(repo SegmentIDGenRepo) *SegmentIdGenUsecase {
	logger := log.NewHelper(log.NewStdLogger(io.Discard))
	return &SegmentIdGenUsecase{
		repo:             repo,
		conf:             &conf.Data{},
		quickLoadSeqFlag: make(chan struct{}, 1),
		log:              logger,
	}
}

func TestUpdateSegmentFromDbAdjustsStep(t *testing.T) {
	repo := &segmentRepoStub{baseStep: 100}
	uc := newTestSegmentUsecase(repo)

	buffer := model.NewSegmentBuffer()
	buffer.SetInitOk(true)
	buffer.SetStep(repo.baseStep)
	buffer.SetMinStep(repo.baseStep)
	buffer.SetUpdateTimeStamp(time.Now().Add(-SegmentDuration / 2).Unix())

	if err := uc.updateSegmentFromDb(context.Background(), "biz", buffer.GetCurrent()); err != nil {
		t.Fatalf("updateSegmentFromDb returned error: %v", err)
	}

	if got := buffer.GetStep(); got != repo.baseStep*2 {
		t.Fatalf("expected step to double to %d, got %d", repo.baseStep*2, got)
	}
	if repo.lastCustomStep != repo.baseStep*2 {
		t.Fatalf("expected repo to be called with step %d, got %d", repo.baseStep*2, repo.lastCustomStep)
	}

	buffer.SetUpdateTimeStamp(time.Now().Add(-(2*SegmentDuration + time.Minute)).Unix())

	if err := uc.updateSegmentFromDb(context.Background(), "biz", buffer.GetCurrent()); err != nil {
		t.Fatalf("updateSegmentFromDb returned error on shrink: %v", err)
	}

	if got := buffer.GetStep(); got != repo.baseStep {
		t.Fatalf("expected step to shrink back to %d, got %d", repo.baseStep, got)
	}
	if repo.lastCustomStep != repo.baseStep {
		t.Fatalf("expected repo to receive shrink step %d, got %d", repo.baseStep, repo.lastCustomStep)
	}
	if repo.customCalls != 2 {
		t.Fatalf("expected UpdateMaxIdByCustomStep to be called twice, got %d", repo.customCalls)
	}
}

func TestLoadSeqsRemovesStaleTags(t *testing.T) {
	repo := &segmentRepoStub{
		baseStep: 100,
		getAllTags: func(ctx context.Context) ([]string, error) {
			return []string{"tag1"}, nil
		},
	}
	uc := newTestSegmentUsecase(repo)

	uc.cache.Store("tag1", model.NewSegmentBuffer())
	uc.cache.Store("tag2", model.NewSegmentBuffer())

	if err := uc.loadSeqs(); err != nil {
		t.Fatalf("loadSeqs returned error: %v", err)
	}

	if _, ok := uc.cache.Load("tag2"); ok {
		t.Fatalf("expected stale tag2 to be removed from cache")
	}
	if _, ok := uc.cache.Load("tag1"); !ok {
		t.Fatalf("expected tag1 to remain in cache")
	}
}
