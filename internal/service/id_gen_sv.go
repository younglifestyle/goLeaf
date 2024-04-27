package service

import (
	"context"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/golang/protobuf/ptypes/timestamp"
	v1 "goLeaf/api/leaf-grpc/v1"
	"goLeaf/internal/biz"
	"goLeaf/internal/biz/model"
	mytime "goLeaf/internal/pkg/time"
	"strconv"
	"time"
)

// IdGenService is a greeter service.
type IdGenService struct {
	v1.UnimplementedLeafSegmentServiceServer
	v1.UnimplementedLeafSnowflakeServiceServer

	segmentIdGenUsecase   *biz.SegmentIdGenUsecase
	snowflakeIdGenUsecase *biz.SnowflakeIdGenUsecase
	log                   *log.Helper
}

// NewIdGenService new a leaf-grpc service.
func NewIdGenService(segmentIdGenUsecase *biz.SegmentIdGenUsecase, snowflakeIdGenUsecase *biz.SnowflakeIdGenUsecase, logger log.Logger) *IdGenService {
	return &IdGenService{
		segmentIdGenUsecase:   segmentIdGenUsecase,
		snowflakeIdGenUsecase: snowflakeIdGenUsecase,
		log:                   log.NewHelper(log.With(logger, "module", "leaf-grpc/service")),
	}
}

func (s *IdGenService) GetServerTimestamp(ctx context.Context, in *v1.GetServerTimestampReq) (ts *v1.GetServerTimestampResp, err error) {

	return &v1.GetServerTimestampResp{
		Timestamp: &timestamp.Timestamp{
			Seconds: time.Now().Unix(),
		}}, nil
}

func (s *IdGenService) GenSnowflakeId(ctx context.Context, in *v1.IdRequest) (idResp *v1.IdReply, err error) {

	id, err := s.snowflakeIdGenUsecase.GetSnowflakeID(ctx)
	if err != nil {
		s.log.Error("get id error : ", err)
		return &v1.IdReply{}, errors.Unwrap(err)
	}

	return &v1.IdReply{Id: id}, nil
}

func (s *IdGenService) DecodeSnowflakeId(ctx context.Context, in *v1.DecodeSnowflakeIdReq) (snowflakeIdResp *v1.DecodeSnowflakeIdResp, err error) {

	snowflakeId, err := strconv.ParseInt(in.Id, 10, 64)
	if err != nil {
		s.log.Error("id error : ", err)
		return nil, biz.ErrSnowflakeIdIllegal
	}

	snowflakeIdResp = &v1.DecodeSnowflakeIdResp{}
	originTimestamp := (snowflakeId >> 22) + 1288834974657
	timeStr := mytime.GetDateTimeStr(mytime.UnixToMS(originTimestamp))
	snowflakeIdResp.Timestamp = strconv.FormatInt(originTimestamp, 10) + "(" + timeStr + ")"

	workerId := (snowflakeId >> 12) ^ (snowflakeId >> 22 << 10)
	snowflakeIdResp.WorkerId = strconv.FormatInt(workerId, 10)

	sequence := snowflakeId ^ (snowflakeId >> 12 << 12)
	snowflakeIdResp.SequenceId = strconv.FormatInt(sequence, 10)

	return
}

func (s *IdGenService) CreateSegmentId(ctx context.Context, leafAllocDb *v1.LeafAllocDb) (resp *v1.CreateSegmentIdResp, err error) {

	resp = &v1.CreateSegmentIdResp{}
	err = s.segmentIdGenUsecase.CreateSegment(ctx, &model.LeafAlloc{
		BizTag:      leafAllocDb.BizTag,
		MaxId:       leafAllocDb.MaxId,
		Step:        int(leafAllocDb.Step),
		Description: leafAllocDb.Description,
		AutoClean:   leafAllocDb.AutoClean,
	})
	if err != nil {
		s.log.Error("create id error : ", err)
		return resp, errors.Unwrap(err)
	}

	return resp, nil
}

func (s *IdGenService) GenSegmentIds(ctx context.Context, genSegmentIdsReq *v1.GenSegmentIdsReq) (*v1.GenSegmentIdsReply, error) {

	resp := &v1.GenSegmentIdsReply{}
	for i := 0; i < int(genSegmentIdsReq.Num); i++ {
		id, err := s.segmentIdGenUsecase.GetSegID(ctx, genSegmentIdsReq.Tag)
		if err != nil {
			s.log.Error("get id error : ", err)
			return &v1.GenSegmentIdsReply{}, errors.Unwrap(err)
		}

		resp.Ids = append(resp.Ids, strconv.FormatInt(id, 10))
	}

	return resp, nil
}

// GenSegmentId
func (s *IdGenService) GenSegmentId(ctx context.Context, idRequest *v1.IdRequest) (*v1.IdReply, error) {

	id, err := s.segmentIdGenUsecase.GetSegID(ctx, idRequest.Tag)
	if err != nil {
		s.log.Error("get id error : ", err)
		return &v1.IdReply{}, errors.Unwrap(err)
	}

	return &v1.IdReply{Id: id}, nil
}

func (s *IdGenService) GenSegmentCache(ctx context.Context,
	idRequest *v1.IdRequest) (segbuffViews *v1.SegmentBufferCacheViews, err error) {
	segbuffViews = &v1.SegmentBufferCacheViews{}

	bufferViews, err := s.segmentIdGenUsecase.Cache(ctx, idRequest.Tag)
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
		segbuff.AutoClean = view.AutoClean
		segbuffViews.SegmentBufferCacheView = append(segbuffViews.SegmentBufferCacheView, segbuff)
	}

	return
}

func (s *IdGenService) GenSegmentDb(ctx context.Context, in *v1.IdRequest) (leafs *v1.LeafAllocDbs, err error) {
	leafs = &v1.LeafAllocDbs{}

	allLeafs, err := s.segmentIdGenUsecase.GetAllLeafs(ctx, in.Tag)
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
		v.AutoClean = leaf.AutoClean
		leafs.LeafAllocDbs = append(leafs.LeafAllocDbs, v)
	}

	return
}
