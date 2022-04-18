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
