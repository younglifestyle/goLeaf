package service

import (
	"context"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	v1 "seg-server/api/leaf-grpc/v1"
	"seg-server/internal/biz"
	"seg-server/internal/util"
	"strconv"
)

// SegmentService is a greeter service.
type SegmentService struct {
	v1.UnimplementedLeafServer

	idGenUc *biz.IDGenUsecase
	log     *log.Helper
}

// NewSegmentService new a leaf-grpc service.
func NewSegmentService(segmentUc *biz.IDGenUsecase, logger log.Logger) *SegmentService {
	return &SegmentService{
		idGenUc: segmentUc,
		log:     log.NewHelper(log.With(logger, "module", "leaf-grpc/service")),
	}
}

func (s *SegmentService) GenSnowflakeId(ctx context.Context, in *v1.IDRequest) (idResp *v1.IDReply, err error) {

	id, err := s.idGenUc.GetSnowflakeID(ctx)
	if err != nil {
		s.log.Error("get id error : ", err)
		return &v1.IDReply{}, errors.Unwrap(err)
	}

	return &v1.IDReply{Id: strconv.FormatInt(id, 10)}, nil
}

func (s *SegmentService) DecodeSnowflakeId(ctx context.Context, in *v1.DecodeSnowflakeIdReq) (snowflakeIdResp *v1.DecodeSnowflakeIdResp, err error) {

	snowflakeId, err := strconv.ParseInt(in.Id, 10, 64)
	if err != nil {
		s.log.Error("id error : ", err)
		return nil, biz.ErrSnowflakeIdIllegal
	}

	snowflakeIdResp = &v1.DecodeSnowflakeIdResp{}
	originTimestamp := (snowflakeId >> 22) + 1288834974657
	timeStr := util.GetDateTimeStr(util.UnixToMS(originTimestamp))
	snowflakeIdResp.Timestamp = strconv.FormatInt(originTimestamp, 10) + "(" + timeStr + ")"

	workerId := (snowflakeId >> 12) ^ (snowflakeId >> 22 << 10)
	snowflakeIdResp.WorkerId = strconv.FormatInt(workerId, 10)

	sequence := snowflakeId ^ (snowflakeId >> 12 << 12)
	snowflakeIdResp.SequenceId = strconv.FormatInt(sequence, 10)

	return
}

// GenSegmentId
func (s *SegmentService) GenSegmentId(ctx context.Context, idRequest *v1.IDRequest) (*v1.IDReply, error) {

	id, err := s.idGenUc.GetSegID(ctx, idRequest.Tag)
	if err != nil {
		s.log.Error("get id error : ", err)
		return &v1.IDReply{}, errors.Unwrap(err)
	}

	return &v1.IDReply{Id: strconv.FormatInt(id, 10)}, nil
}

func (s *SegmentService) GenSegmentCache(ctx context.Context,
	idRequest *v1.IDRequest) (segbuffViews *v1.SegmentBufferCacheViews, err error) {
	segbuffViews = &v1.SegmentBufferCacheViews{}

	bufferViews, err := s.idGenUc.Cache(ctx)
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

	allLeafs, err := s.idGenUc.GetAllLeafs(ctx)
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
		v.UpdatedTime = leaf.UpdatedAt.Unix()
		v.CreatedTime = leaf.CreatedAt.Unix()
		leafs.LeafAllocDbs = append(leafs.LeafAllocDbs, v)
	}

	return
}
