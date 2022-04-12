package service

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"seg-server/internal/data"
	"seg-server/internal/model"
	"sync"

	v1 "seg-server/api/helloworld/v1"
)

// SegmentService is a greeter service.
type SegmentService struct {
	v1.UnimplementedGreeterServer

	data     *data.Data
	ls       map[string]*model.LeafAlloc
	leafLock sync.RWMutex
	log      *log.Helper
}

// NewSegmentService new a segment service.
func NewSegmentService(data *data.Data, logger log.Logger) *SegmentService {
	return &SegmentService{
		data: data,
		ls:   make(map[string]*model.LeafAlloc),
		log:  log.NewHelper(log.With(logger, "module", "segment/service")),
	}
}

// SayHello implements helloworld.GreeterServer.
func (s *SegmentService) SayHello(ctx context.Context, in *v1.HelloRequest) (*v1.HelloReply, error) {
	return &v1.HelloReply{Message: "Hello "}, nil
}

// GenSegmentId
func (s *SegmentService) GenSegmentId(ctx context.Context, in *v1.IDRequest) (*v1.IDReply, error) {
	return &v1.IDReply{Message: "Hello "}, nil
}
