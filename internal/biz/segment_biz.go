package biz

import (
	"context"
	"fmt"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"go.uber.org/atomic"
	"golang.org/x/sync/singleflight"
	v1 "seg-server/api/leaf-grpc/v1"
	"seg-server/internal/biz/model"
	"sync"
	"time"
)

var (
	ErrDBOps = errors.NotFound(v1.ErrorReason_DB_OPERATE.String(), "update and get id error")
	// ErrTagNotFound key不存在时的异常码
	ErrTagNotFound          = errors.NotFound(v1.ErrorReason_BIZ_TAG_NOT_FOUND.String(), "biz tag not found")
	ErrIDCacheInitFalse     = errors.InternalServer(v1.ErrorReason_IDCacheInitFalse.String(), "id cache init false")
	ErrIDTwoSegmentsAreNull = errors.InternalServer(v1.ErrorReason_IDTwoSegmentsAreNull.String(), "id two segments are null")
)

const (
	SegmentDuration = time.Minute * 15
	MaxStep         = 1000000
)

// SegmentRepo is a Greater repo.
type SegmentRepo interface {
	UpdateAndGetMaxId(ctx context.Context, tag string) (seg model.LeafAlloc, err error)
	GetLeafAlloc(ctx context.Context, tag string) (seg model.LeafAlloc, err error)
	GetAllTags(ctx context.Context) (tags []string, err error)
	GetAllLeafAllocs(ctx context.Context) (leafs []*model.LeafAlloc, err error)
	UpdateMaxIdByCustomStepAndGetLeafAlloc(ctx context.Context, tag string, step int) (leafAlloc model.LeafAlloc, err error)
}

// SegmentUsecase is a Segment usecase.
type SegmentUsecase struct {
	repo            SegmentRepo
	singleGroup     singleflight.Group
	segmentDuration int64    // 号段消耗时间
	cache           sync.Map // biz-tag : model.SegmentBuffer

	log *log.Helper
}

// NewSegmentUsecase new a Segment usecase.
func NewSegmentUsecase(repo SegmentRepo, logger log.Logger) *SegmentUsecase {
	s := &SegmentUsecase{
		repo: repo,
		log:  log.NewHelper(log.With(logger, "module", "leaf-grpc/biz"))}

	s.loadSeqs()
	go s.loadProc()

	return s
}

// GetID creates a Segment, and returns the new Segment.
func (uc *SegmentUsecase) GetID(ctx context.Context, tag string) (int64, error) {

	value, ok := uc.cache.Load(tag)
	if !ok {
		return 0, ErrTagNotFound
	}
	segmentBuffer := value.(*model.SegmentBuffer)
	if !segmentBuffer.IsInitOk() {
		_, err, _ := uc.singleGroup.Do(tag, func() (res interface{}, err error) {
			if !segmentBuffer.IsInitOk() {
				err := uc.updateSegmentFromDb(ctx, tag, segmentBuffer.GetCurrent())
				if err != nil {
					segmentBuffer.SetInitOk(false)
					return 0, err
				}
				segmentBuffer.SetInitOk(true)
			}
			return
		})
		if err != nil {
			return 0, err
		}
	}

	return uc.getIdFromSegmentBuffer(ctx, segmentBuffer)
}

func (uc *SegmentUsecase) updateSegmentFromDb(ctx context.Context, bizTag string, segment *model.Segment) (err error) {

	var leafAlloc model.LeafAlloc

	segmentBuffer := segment.GetBuffer()

	// 如果buffer没有DB数据初始化(也就是第一次进行DB数据初始化)
	if !segmentBuffer.IsInitOk() {
		leafAlloc, err = uc.repo.UpdateAndGetMaxId(ctx, bizTag)
		if err != nil {
			uc.log.Error("db error : ", err)
			return fmt.Errorf("db error : %s %w", err, ErrDBOps)
		}
		segmentBuffer.SetStep(leafAlloc.Step)
		segmentBuffer.SetMinStep(leafAlloc.Step)
	} else if segmentBuffer.GetUpdateTimeStamp() == 0 {
		// 如果buffer的更新时间是0（初始是0，也就是第二次调用updateSegmentFromDb()）
		leafAlloc, err = uc.repo.UpdateAndGetMaxId(ctx, bizTag)
		if err != nil {
			uc.log.Error("db error : ", err)
			return fmt.Errorf("db error : %s %w", err, ErrDBOps)
		}
		segmentBuffer.SetUpdateTimeStamp(time.Now().Unix())
		segmentBuffer.SetMinStep(leafAlloc.Step)
	} else {
		// 第三次以及之后的进来 动态设置nextStep
		// 计算当前更新操作和上一次更新时间差
		duration := time.Now().Unix() - segmentBuffer.GetUpdateTimeStamp()
		nextStep := segmentBuffer.GetStep()
		/**
		 *  动态调整step
		 *  1) duration < 15 分钟 : step 变为原来的2倍， 最大为 MAX_STEP
		 *  2) 15分钟 <= duration < 30分钟 : nothing
		 *  3) duration >= 30 分钟 : 缩小step, 最小为DB中配置的step
		 *
		 *  这样做的原因是认为15min一个号段大致满足需求
		 *  如果updateSegmentFromDb()速度频繁(15min多次)，也就是
		 *  如果15min这个时间就把step号段用完，为了降低数据库访问频率，
		 *  我们可以扩大step大小，相反如果将近30min才把号段内的id用完，则可以缩小step
		 */
		// duration < 15 分钟 : step 变为原来的2倍. 最大为 MAX_STEP
		if duration < int64(SegmentDuration) {
			if nextStep*2 > MaxStep {
				//do nothing
			} else {
				// 步数 * 2
				nextStep = nextStep * 2
			}
		} else if duration < int64(SegmentDuration)*2 {
			// 15分钟 < duration < 30分钟 : nothing
		} else {
			// duration > 30 分钟 : 缩小step, 最小为DB中配置的步数
			if nextStep/2 >= segmentBuffer.GetMinStep() {
				nextStep = nextStep / 2
			}
		}
		leafAlloc, err = uc.repo.UpdateMaxIdByCustomStepAndGetLeafAlloc(ctx, bizTag, nextStep)
		if err != nil {
			uc.log.Error("custom db error : ", err)
			return fmt.Errorf("custom db error : %s %w", err, ErrDBOps)
		}
		segmentBuffer.SetUpdateTimeStamp(time.Now().Unix())
		segmentBuffer.SetStep(nextStep)
		segmentBuffer.SetMinStep(leafAlloc.Step)
	}

	value := leafAlloc.MaxId - int64(segmentBuffer.GetStep())
	segment.GetValue().Store(value)
	segment.SetMax(leafAlloc.MaxId)
	segment.SetStep(segmentBuffer.GetStep())

	return
}

func (uc *SegmentUsecase) updateAnotherSegmentFromDb(ctx context.Context, bizTag string, segmentBuffer *model.SegmentBuffer) (err error) {

	uc.log.Info("updateAnotherSegmentFromDb ....")

	var leafAlloc model.LeafAlloc

	leafAlloc, err = uc.repo.UpdateAndGetMaxId(ctx, bizTag)
	if err != nil {
		uc.log.Error("db error : ", err)
		return fmt.Errorf("db error : %s %w", err, ErrDBOps)
	}
	segmentBuffer.SetStep(leafAlloc.Step)

	value := leafAlloc.MaxId - int64(segmentBuffer.GetStep())
	segment := segmentBuffer.GetSegments()[segmentBuffer.NextPos()]
	segment.GetValue().Store(value)
	segment.SetMax(leafAlloc.MaxId)
	segment.SetStep(segmentBuffer.GetStep())

	uc.log.Info("updateSegmentFromDb value : ", value, leafAlloc.MaxId)
	uc.log.Infof("updateAnotherSegmentFromDb pointer : %p %p",
		segmentBuffer.Segments[1], segment)
	uc.log.Infof("info Next Current index : %+v",
		*segmentBuffer.Segments[1])

	return
}

func (uc *SegmentUsecase) loadNextSegmentFromDb(ctx context.Context, cacheSegmentBuffer *model.SegmentBuffer) {
	segment := cacheSegmentBuffer.GetSegments()[cacheSegmentBuffer.NextPos()]
	err := uc.updateSegmentFromDb(ctx, cacheSegmentBuffer.GetKey(), segment)
	if err != nil {
		cacheSegmentBuffer.GetThreadRunning().Store(false)
	}

	cacheSegmentBuffer.WLock()
	defer cacheSegmentBuffer.WUnLock()
	cacheSegmentBuffer.SetNextReady(true)
	cacheSegmentBuffer.GetThreadRunning().Store(false)

	return
}

func waitAndSleep(segmentBuffer *model.SegmentBuffer) {
	roll := 0
	for segmentBuffer.GetThreadRunning().Load() {
		roll++
		if roll > 10000 {
			time.Sleep(time.Duration(10) * time.Millisecond)
			break
		}
	}
}

func (uc *SegmentUsecase) getIdFromSegmentBuffer(ctx context.Context, cacheSegmentBuffer *model.SegmentBuffer) (int64, error) {

	var (
		segment *model.Segment
		value   int64
		err     error
	)

	for {
		if value := func() int64 {
			cacheSegmentBuffer.RLock()
			defer cacheSegmentBuffer.RUnLock()

			segment = cacheSegmentBuffer.GetCurrent()
			if !cacheSegmentBuffer.IsNextReady() &&
				(segment.GetIdle() < int64(0.9*float64(segment.GetStep()))) &&
				cacheSegmentBuffer.GetThreadRunning().CAS(false, true) {

				go uc.loadNextSegmentFromDb(context.TODO(), cacheSegmentBuffer)
			}

			value = segment.GetValue().Load()
			segment.GetValue().Inc()
			if value < segment.GetMax() { // 成功返回
				return value
			}
			return 0
		}(); value != 0 {
			return value, nil
		}

		// 等待协程异步准备号段完毕
		waitAndSleep(cacheSegmentBuffer)

		value, err = func() (int64, error) {
			// 执行到这里，说明当前号段已经用完，应该切换另一个Segment号段使用
			cacheSegmentBuffer.WLock()
			defer cacheSegmentBuffer.WUnLock()

			// 重复获取value, 并发执行时，Segment可能已经被其他协程切换。再次判断, 防止重复切换Segment
			segment = cacheSegmentBuffer.GetCurrent()
			value = segment.GetValue().Load()
			segment.GetValue().Inc()
			if value < segment.GetMax() { // 成功返回
				return value, nil
			}

			// 执行到这里, 说明其他的协程没有进行Segment切换，
			// 并且当前号段所有号码用完，需要进行切换Segment
			// 如果准备好另一个Segment，直接切换
			if cacheSegmentBuffer.IsNextReady() {
				cacheSegmentBuffer.SwitchPos()
				cacheSegmentBuffer.SetNextReady(false)
			} else { // 如果另一个Segment没有准备好，则返回异常双buffer全部用完
				return 0, ErrIDTwoSegmentsAreNull
			}
			return 0, nil
		}()
		if value != 0 || err != nil {
			return value, err
		}
	}
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

	bizTags, err := uc.repo.GetAllTags(context.TODO())
	if err != nil {
		log.Error("load tags error : ", err)
		return err
	}
	if len(bizTags) == 0 {
		return
	}

	// 数据库中的tag
	insertTags := []string{}
	removeTags := []string{}
	// 当前的cache中所有的tag
	cacheTags := map[string]struct{}{}
	uc.cache.Range(func(k, v interface{}) bool {
		cacheTags[k.(string)] = struct{}{}
		return true
	})

	// 下面两步操作：保证cache和数据库tags同步
	// 1. db中新加的tags灌进cache，并实例化初始对应的SegmentBuffer
	for _, k := range bizTags {
		if _, ok := cacheTags[k]; !ok {
			insertTags = append(insertTags, k)
		}
	}
	for _, k := range insertTags {
		segmentBuffer := model.NewSegmentBuffer()
		segmentBuffer.SetKey(k)
		segment := segmentBuffer.GetCurrent()
		segment.SetValue(atomic.NewInt64(0))
		segment.SetMax(0)
		segment.SetStep(0)
		uc.cache.Store(k, segmentBuffer)
		uc.log.Infof("Add tag {%s} from db to IdCache", k)
	}

	// 2. cache中已失效的tags从cache删除
	for _, k := range bizTags {
		if _, ok := cacheTags[k]; !ok {
			removeTags = append(removeTags, k)
		}
	}
	if len(removeTags) > 0 && len(cacheTags) > 0 {
		for _, tag := range removeTags {
			uc.cache.Delete(tag)
		}
	}

	return nil
}
