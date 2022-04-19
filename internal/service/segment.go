package service

import (
	"context"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	v1 "seg-server/api/leaf-grpc/v1"
	"seg-server/internal/biz"
	"strconv"
)

// SegmentService is a greeter service.
type SegmentService struct {
	v1.UnimplementedLeafServer

	segmentUc *biz.SegmentUsecase
	log       *log.Helper
}

// NewSegmentService new a leaf-grpc service.
func NewSegmentService(segmentUc *biz.SegmentUsecase, logger log.Logger) *SegmentService {
	return &SegmentService{
		segmentUc: segmentUc,
		log:       log.NewHelper(log.With(logger, "module", "leaf-grpc/service")),
	}
}

// GenSegmentId
func (s *SegmentService) GenSegmentId(ctx context.Context, idRequest *v1.IDRequest) (*v1.IDReply, error) {

	id, err := s.segmentUc.GetID(ctx, idRequest.Tag)
	if err != nil {
		s.log.Error("get id error : ", err)
		return &v1.IDReply{}, errors.Unwrap(err)
	}

	return &v1.IDReply{Id: strconv.FormatInt(id, 10)}, nil
}

func (s *SegmentService) GenSegmentCache(ctx context.Context,
	idRequest *v1.IDRequest) (segbuffViews *v1.SegmentBufferCacheViews, err error) {
	segbuffViews = &v1.SegmentBufferCacheViews{}

	bufferViews, err := s.segmentUc.Cache(ctx)
	if err != nil {
		s.log.Error("get segment cache error : ", err)
		return
	}
	for _, view := range bufferViews {
		segbuff := &v1.SegmentBufferCacheView{}
		segbuff.InitOk = view.InitOk
		segbuff.Key = view.Key
		segbuff.Pos = int32(view.Pos)
		segbuff.NextReady = view.NextReady
		segbuff.Max0 = view.Max0
		segbuff.Value0 = view.Value0
		segbuff.Step0 = int32(view.Step0)
		segbuff.Max1 = view.Max1
		segbuff.Value1 = view.Value1
		segbuff.Step1 = int32(view.Step1)
		segbuffViews.SegmentBufferCacheView = append(segbuffViews.SegmentBufferCacheView, segbuff)
	}

	return
}

func (s *SegmentService) GenSegmentDB(ctx context.Context, in *v1.IDRequest) (leafs *v1.LeafAllocDbs, err error) {
	leafs = &v1.LeafAllocDbs{}

	allLeafs, err := s.segmentUc.GetAllLeafs(ctx)
	if err != nil {
		s.log.Error("get segment db error : ", err)
		return
	}

	for _, leaf := range allLeafs {
		v := &v1.LeafAllocDb{}
		v.BizTag = leaf.BizTag
		v.MaxId = leaf.MaxId
		v.Step = int32(leaf.Step)
		v.Description = leaf.Description
		v.UpdatedTime = leaf.UpdatedAt
		v.CreatedTime = leaf.CreatedAt
		leafs.LeafAllocDbs = append(leafs.LeafAllocDbs, v)
	}

	return
}
