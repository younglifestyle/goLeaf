package biz

import (
	"context"
	"fmt"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/robfig/cron"
	"go.uber.org/atomic"
	v1 "goLeaf/api/leaf-grpc/v1"
	"goLeaf/internal/biz/model"
	"goLeaf/internal/conf"
	"golang.org/x/sync/singleflight"
	"sync"
	"time"
)

var (
	ErrIDExist = errors.BadRequest(v1.ErrorReason_DB_OPERATE.String(), "")
	ErrDBOps   = errors.NotFound(v1.ErrorReason_DB_OPERATE.String(), "update and get id error")
	// ErrTagNotFound key不存在时的异常码
	ErrTagNotFound            = errors.NotFound(v1.ErrorReason_BIZ_TAG_NOT_FOUND.String(), "biz tag not found")
	ErrIDCacheInitFalse       = errors.InternalServer(v1.ErrorReason_ID_CACHE_INIT_FALSE.String(), "id cache init false")
	ErrIDTwoSegmentsAreNull   = errors.InternalServer(v1.ErrorReason_ID_TWO_SEGMENTS_ARE_NULL.String(), "id two segments are null")
	ErrSnowflakeTimeException = errors.InternalServer(v1.ErrorReason_SNOWFLAKE_TIME_EXCEPTION.String(), "time callback")
	ErrSnowflakeIdIllegal     = errors.BadRequest(v1.ErrorReason_SNOWFLAKE_ID_ILLEGAL.String(), "id illegal")
)

const (
	SegmentDuration = time.Minute * 15
	MaxStep         = 1000000
)

// SegmentIDGenRepo DB and Etcd operate sets
type SegmentIDGenRepo interface {
	SaveLeafAlloc(ctx context.Context, leafAlloc *model.LeafAlloc) error

	UpdateAndGetMaxId(ctx context.Context, tag string) (seg model.LeafAlloc, err error)
	GetLeafAlloc(ctx context.Context, tag string) (seg model.LeafAlloc, err error)
	GetAllTags(ctx context.Context) (tags []string, err error)
	GetAllLeafAllocs(ctx context.Context) (leafs []*model.LeafAlloc, err error)
	UpdateMaxIdByCustomStepAndGetLeafAlloc(ctx context.Context, tag string, step int) (leafAlloc model.LeafAlloc, err error)

	CleanLeafMaxId(ctx context.Context, tags []string) (err error)
}

// SegmentIdGenUsecase is a Segment usecase.
type SegmentIdGenUsecase struct {
	repo             SegmentIDGenRepo
	conf             *conf.Data
	singleGroup      singleflight.Group
	segmentDuration  int64    // 号段消耗时间
	cache            sync.Map // k biz-tag : v model.SegmentBuffer
	quickLoadSeqFlag chan struct{}

	twepoch       int64
	workerId      int64
	sequence      int64
	lastTimestamp int64
	snowFlakeLock sync.Mutex

	model.SnowFlakeEtcdHolder

	log *log.Helper
}

// NewSegmentIdGenUsecase new a Segment usecase.
func NewSegmentIdGenUsecase(repo SegmentIDGenRepo, conf *conf.Bootstrap, logger log.Logger) *SegmentIdGenUsecase {
	s := &SegmentIdGenUsecase{
		repo:             repo,
		conf:             conf.Data,
		quickLoadSeqFlag: make(chan struct{}),
		log:              log.NewHelper(log.With(logger, "module", "leaf-grpc/segment"))}

	// 号段模式启用时，启动
	if conf.Data.Database.SegmentEnable {
		_ = s.loadSeqs()
		s.autoCleanProc()
		go s.loadProc()
	}

	return s
}

func (uc *SegmentIdGenUsecase) GetAllLeafs(ctx context.Context) ([]*model.LeafAlloc, error) {
	if uc.conf.Database.SegmentEnable {
		return uc.repo.GetAllLeafAllocs(ctx)
	} else {
		return nil, nil
	}
}

func (uc *SegmentIdGenUsecase) Cache(ctx context.Context) ([]*model.SegmentBufferView, error) {
	bufferViews := []*model.SegmentBufferView{}
	uc.cache.Range(func(k, v interface{}) bool {
		segBuff := v.(*model.SegmentBuffer)
		bv := &model.SegmentBufferView{}
		bv.InitOk = segBuff.IsInitOk()
		bv.Key = segBuff.GetKey()
		bv.Pos = segBuff.GetCurrentPos()
		bv.NextReady = segBuff.IsNextReady()
		bv.AutoClean = segBuff.AutoClean
		segments := segBuff.GetSegments()
		bv.Max0 = segments[0].GetMax()
		bv.Value0 = segments[0].GetValue().Load()
		bv.Step0 = segments[0].GetStep()
		bv.Max1 = segments[1].GetMax()
		bv.Value1 = segments[1].GetValue().Load()
		bv.Step1 = segments[1].GetStep()
		bufferViews = append(bufferViews, bv)
		return true
	})

	return bufferViews, nil
}

// GetSegID creates a Segment, and returns the new Segment.
func (uc *SegmentIdGenUsecase) GetSegID(ctx context.Context, tag string) (int64, error) {
	if uc.conf.Database.SegmentEnable {
		value, ok := uc.cache.Load(tag)
		if !ok {
			return 0, ErrTagNotFound
		}
		segmentBuffer := value.(*model.SegmentBuffer)
		if !segmentBuffer.IsInitOk() {
			// 此处用锁效果一致
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
				return -1, err
			}
		}

		return uc.getIdFromSegmentBuffer(ctx, segmentBuffer)
	} else {
		return -1, nil
	}
}

func (uc *SegmentIdGenUsecase) CreateSegment(ctx context.Context, leafAlloc *model.LeafAlloc) (err error) {

	err = uc.repo.SaveLeafAlloc(ctx, leafAlloc)
	if err != nil {
		return err
	}

	go func() {
		// 快速load号段，以使号段添加进入缓存
		uc.quickLoadSeqFlag <- struct{}{}
	}()

	return nil
}

func (uc *SegmentIdGenUsecase) updateSegmentFromDb(ctx context.Context, bizTag string, segment *model.Segment) (err error) {

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
		segmentBuffer.SetAutoClean(leafAlloc.AutoClean)
	} else if segmentBuffer.GetUpdateTimeStamp() == 0 {
		// 如果buffer的更新时间是0（初始是0，也就是第二次调用updateSegmentFromDb()）
		leafAlloc, err = uc.repo.UpdateAndGetMaxId(ctx, bizTag)
		if err != nil {
			uc.log.Error("db error : ", err)
			return fmt.Errorf("db error : %s %w", err, ErrDBOps)
		}
		segmentBuffer.SetUpdateTimeStamp(time.Now().Unix())
		segmentBuffer.SetMinStep(leafAlloc.Step)
		segmentBuffer.SetAutoClean(leafAlloc.AutoClean)
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
		segmentBuffer.SetAutoClean(leafAlloc.AutoClean)
	}

	value := leafAlloc.MaxId - int64(segmentBuffer.GetStep())
	segment.GetValue().Store(value)
	segment.SetMax(leafAlloc.MaxId)
	segment.SetStep(segmentBuffer.GetStep())

	return
}

func (uc *SegmentIdGenUsecase) loadNextSegmentFromDb(ctx context.Context, cacheSegmentBuffer *model.SegmentBuffer) {
	segment := cacheSegmentBuffer.GetSegments()[cacheSegmentBuffer.NextPos()]
	err := uc.updateSegmentFromDb(ctx, cacheSegmentBuffer.GetKey(), segment)
	if err != nil {
		cacheSegmentBuffer.GetThreadRunning().Store(false)
		return
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

func (uc *SegmentIdGenUsecase) getIdFromSegmentBuffer(ctx context.Context, cacheSegmentBuffer *model.SegmentBuffer) (int64, error) {

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

				// 协程中传入空ctx，防止主体执行完成后其调用cancel取消上下文
				go uc.loadNextSegmentFromDb(context.TODO(), cacheSegmentBuffer)
			}

			value = segment.GetValue().Load()
			segment.GetValue().Inc()
			if value < segment.GetMax() { // 成功返回
				return value
			}
			return -1
		}(); value != -1 {
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
				return -1, ErrIDTwoSegmentsAreNull
			}
			return -1, nil
		}()
		if value != -1 || err != nil {
			return value, err
		}
	}
}

// loadProc 定时1min同步一次db和cache
func (uc *SegmentIdGenUsecase) loadProc() {
	ticker := time.NewTicker(time.Minute)

	for {
		select {
		case <-ticker.C:
			_ = uc.loadSeqs()
		case <-uc.quickLoadSeqFlag:
			_ = uc.loadSeqs()
			ticker.Reset(time.Minute)
		}
	}
}

// 添加一个每天 0 点 0 分 0 秒执行的清零定时任务  0 0 0 * * *  0/5 * * * * *
func (uc *SegmentIdGenUsecase) autoCleanProc() {
	c := cron.New()
	err := c.AddFunc("0 0 0 * * *", func() {
		_ = uc.autoCleanSeqs()
	})
	if err != nil {
		log.Errorf("Failed to add auto clean cron job : ", err)
		return
	}
	c.Start()
}

func (uc *SegmentIdGenUsecase) loadSeqs() (err error) {

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
		cacheTags[k] = struct{}{}
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

func (uc *SegmentIdGenUsecase) autoCleanSeqs() (err error) {

	var needCleanBizTags []string

	uc.cache.Range(func(k, v interface{}) bool {
		segmentBuffer := v.(*model.SegmentBuffer)
		if !segmentBuffer.AutoClean {
			return true
		}

		needCleanBizTags = append(needCleanBizTags, segmentBuffer.BizKey)
		func() {
			segmentBuffer.WLock()
			defer segmentBuffer.WUnLock()

			segmentBuffer.SetNextReady(false)
			segmentBuffer.Segments[0].Value = atomic.NewInt64(1)
			segmentBuffer.Segments[0].MaxId = int64(segmentBuffer.GetStep() + 1)
		}()

		return true
	})

	if len(needCleanBizTags) != 0 {
		err = uc.repo.CleanLeafMaxId(context.Background(), needCleanBizTags)
		if err != nil {
			log.Error("update tags max id to start id error : ", err)
			return err
		}
	}

	return nil
}
