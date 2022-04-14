package service

import (
	"context"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	v1 "seg-server/api/segment/v1"
	"seg-server/internal/biz"
	"strconv"
)

// SegmentService is a greeter service.
type SegmentService struct {
	v1.UnimplementedGreeterServer

	segmentUc *biz.SegmentUsecase
	log       *log.Helper
}

// NewSegmentService new a segment service.
func NewSegmentService(segmentUc *biz.SegmentUsecase, logger log.Logger) *SegmentService {
	return &SegmentService{
		segmentUc: segmentUc,
		log:       log.NewHelper(log.With(logger, "module", "segment/service")),
	}
}

// GenSegmentId
func (s *SegmentService) GenSegmentId(ctx context.Context, idRequest *v1.IDRequest) (*v1.IDReply, error) {

	id, err := s.segmentUc.GetID(ctx, idRequest.Tag)
	if err != nil {
		s.log.Error("get id error : ", err)
		return nil, errors.Unwrap(err)
	}

	return &v1.IDReply{Id: strconv.FormatInt(id, 10)}, nil
}
