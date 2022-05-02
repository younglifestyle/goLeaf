package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/spf13/cast"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/atomic"
	"golang.org/x/sync/singleflight"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	v1 "seg-server/api/leaf-grpc/v1"
	"seg-server/internal/biz/model"
	"seg-server/internal/conf"
	"seg-server/internal/util"
	"strings"
	"sync"
	"time"
)

var (
	ErrDBOps = errors.NotFound(v1.ErrorReason_DB_OPERATE.String(), "update and get id error")
	// ErrTagNotFound key不存在时的异常码
	ErrTagNotFound            = errors.NotFound(v1.ErrorReason_BIZ_TAG_NOT_FOUND.String(), "biz tag not found")
	ErrIDCacheInitFalse       = errors.InternalServer(v1.ErrorReason_ID_CACHE_INIT_FALSE.String(), "id cache init false")
	ErrIDTwoSegmentsAreNull   = errors.InternalServer(v1.ErrorReason_ID_TWO_SEGMENTS_ARE_NULL.String(), "id two segments are null")
	ErrSnowflakeTimeException = errors.InternalServer(v1.ErrorReason_SNOWFLAKE_TIME_EXCEPTION.String(), "time callback")
	ErrSnowflakeIdIllegal     = errors.BadRequest(v1.ErrorReason_SNOWFLAKE_ID_ILLEGAL.String(), "id illegal")

	// 起始的时间戳，用于用当前时间戳减去这个时间戳，算出偏移量
	twepoch            = int64(1288834974657)
	workerIdBits       = 10
	maxWorkerId        = ^(-1 << workerIdBits) // 1023
	sequenceBits       = 12
	workerIdShift      = sequenceBits
	timestampLeftShift = sequenceBits + workerIdBits
	sequenceMask       = int64(^(-1 << sequenceBits)) //4095 0xFFF

	PrefixEtcdPath = "/snowflake/"
	PropPath       string
	PathForever    = PrefixEtcdPath + "/forever" //保存所有数据持久的节点

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

	GetPrefixKey(ctx context.Context, prefix string) (*clientv3.GetResponse, error)
	CreateKeyWithOptLock(ctx context.Context, key string, val string) bool
	CreateOrUpdateKey(ctx context.Context, key string, val string) bool
	GetKey(ctx context.Context, key string) (*clientv3.GetResponse, error)
}

// SegmentUsecase is a Segment usecase.
type SegmentUsecase struct {
	repo            SegmentRepo
	conf            *conf.Data
	singleGroup     singleflight.Group
	segmentDuration int64    // 号段消耗时间
	cache           sync.Map // k biz-tag : v model.SegmentBuffer

	twepoch       int64
	workerId      int64
	sequence      int64
	lastTimestamp int64

	model.SnowFlakeEtcdHolder

	log *log.Helper
}

// NewSegmentUsecase new a Segment usecase.
func NewSegmentUsecase(repo SegmentRepo, conf *conf.Bootstrap, logger log.Logger) *SegmentUsecase {
	s := &SegmentUsecase{
		repo: repo,
		conf: conf.Data,
		log:  log.NewHelper(log.With(logger, "module", "leaf-grpc/biz"))}

	// 号段模式启用时，启动
	if conf.Data.Database.SegmentEnable {
		s.loadSeqs()
		go s.loadProc()
	}

	if conf.Data.Etcd.SnowflakeEnable {
		s.twepoch = twepoch
		if !(time.Now().UnixMilli() > twepoch) {
			panic("Snowflake not support twepoch gt currentTime")
		}

		s.SnowFlakeEtcdHolder.Ip = util.GetOutboundIP()
		s.SnowFlakeEtcdHolder.Port = strings.Split(conf.Server.Http.Addr, ":")[1]

		PrefixEtcdPath = "/snowflake/" + conf.Server.ServerName
		PropPath = filepath.Join(util.GetCurrentAbPath(), conf.Server.ServerName) +
			"/leafconf/" + s.SnowFlakeEtcdHolder.Port + "/workerID.toml"
		PathForever = PrefixEtcdPath + "/forever"

		s.log.Info("PropPath : ", PropPath)

		s.SnowFlakeEtcdHolder.ListenAddress = s.SnowFlakeEtcdHolder.Ip + ":" + s.SnowFlakeEtcdHolder.Port
		if !s.initSnowFlake() {
			s.log.Fatal("Snowflake Service Init Fail")
		} else {
			s.workerId = int64(s.SnowFlakeEtcdHolder.WorkerId)
		}
		if !(s.workerId >= 0 && s.workerId <= int64(maxWorkerId)) {
			panic("workerID must gte 0 and lte 1023")
		}
	}

	return s
}

func (uc *SegmentUsecase) GetAllLeafs(ctx context.Context) ([]*model.LeafAlloc, error) {
	if uc.conf.Database.SegmentEnable {
		return uc.repo.GetAllLeafAllocs(ctx)
	} else {
		return nil, nil
	}
}

func (uc *SegmentUsecase) Cache(ctx context.Context) ([]*model.SegmentBufferView, error) {
	bufferViews := []*model.SegmentBufferView{}
	uc.cache.Range(func(k, v interface{}) bool {
		segBuff := v.(*model.SegmentBuffer)
		bv := &model.SegmentBufferView{}
		bv.InitOk = segBuff.IsInitOk()
		bv.Key = segBuff.GetKey()
		bv.Pos = segBuff.GetCurrentPos()
		bv.NextReady = segBuff.IsNextReady()
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
func (uc *SegmentUsecase) GetSegID(ctx context.Context, tag string) (int64, error) {
	if uc.conf.Database.SegmentEnable {
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
	} else {
		return 0, nil
	}
}

// GetSnowflakeID creates a Snowflake ID  运行时，leaf允许最多5ms的回拨；重启时，允许最多3s的回拨
func (uc *SegmentUsecase) GetSnowflakeID(ctx context.Context) (int64, error) {
	var ts = time.Now().UnixMilli()

	if ts > uc.lastTimestamp {
		offset := uc.lastTimestamp - ts
		if offset <= 5 {
			// 等待 2*offset ms就可以唤醒重新尝试获取锁继续执行
			time.Sleep(time.Duration(offset<<1) * time.Millisecond)
			// 重新获取当前时间戳，理论上这次应该比上一次记录的时间戳迟了
			ts = time.Now().UnixMilli()
			if ts < uc.lastTimestamp {
				return 0, ErrSnowflakeTimeException
			}
		} else {
			return 0, ErrSnowflakeTimeException
		}
	}

	// 如果从上一个逻辑分支产生的timestamp仍然和lastTimestamp相等
	if uc.lastTimestamp == ts {
		// 自增序列+1然后取后12位的值
		uc.sequence = (uc.sequence + 1) & sequenceMask
		// seq 为0的时候表示当前毫秒12位自增序列用完了，应该用下一毫秒时间来区别，否则就重复了
		if uc.sequence == 0 {
			// 对seq做随机作为起始
			uc.sequence = int64(rand.Intn(100))
			// 生成比lastTimestamp滞后的时间戳，这里不进行wait，因为很快就能获得滞后的毫秒数
			ts = util.UtilNextMillis(uc.lastTimestamp)
		}
	} else {
		uc.sequence = int64(rand.Intn(100))
	}
	uc.lastTimestamp = ts
	id := ((ts - uc.twepoch) << timestampLeftShift) | (uc.workerId << workerIdShift) | uc.sequence

	return id, nil
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

func (uc *SegmentUsecase) loadNextSegmentFromDb(ctx context.Context, cacheSegmentBuffer *model.SegmentBuffer) {
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

				// 协程中传入空ctx，防止主体执行完成后其调用cancel取消上下文
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

func (uc *SegmentUsecase) initSnowFlake() bool {
	var retryCount = 0
RETRY:
	prefixKeyResps, err := uc.repo.GetPrefixKey(context.TODO(), PathForever)
	if err == nil {
		// 还没有实例化过
		if prefixKeyResps.Count == 0 {
			uc.SnowFlakeEtcdHolder.EtcdAddressNode = PathForever + "/" + uc.ListenAddress + "-0"
			if success := uc.repo.CreateKeyWithOptLock(context.TODO(),
				uc.SnowFlakeEtcdHolder.EtcdAddressNode,
				string(uc.buildData())); !success {
				// 其他实例已创建
				if retryCount > 3 {
					return false
				}
				retryCount++
				goto RETRY
			}
			uc.updateLocalWorkerID(uc.WorkerId)
			go uc.scheduledUploadData(uc.SnowFlakeEtcdHolder.EtcdAddressNode)
		} else {
			// 存在的话，说明不是第一次启动leaf应用，etcd存在以前的【自身节点标识和时间数据】
			// 自身节点ip:port->1
			nodeMap := make(map[string]int, 0)
			// 自身节点ip:port-> path/ip:port-1
			realNodeMap := make(map[string]string, 0)

			for _, node := range prefixKeyResps.Kvs {
				nodeKey := strings.Split(filepath.Base(string(node.Key)), "-")
				realNodeMap[nodeKey[0]] = string(node.Key)
				nodeMap[nodeKey[0]] = cast.ToInt(nodeKey[1])
			}

			if workId, ok := nodeMap[uc.SnowFlakeEtcdHolder.ListenAddress]; ok {
				uc.SnowFlakeEtcdHolder.EtcdAddressNode = realNodeMap[uc.SnowFlakeEtcdHolder.ListenAddress]
				uc.WorkerId = workId
				if !uc.checkInitTimeStamp(uc.SnowFlakeEtcdHolder.EtcdAddressNode) {
					uc.log.Error("init timestamp check error,forever node timestamp gt this node time")
					return false
				}

				go uc.scheduledUploadData(uc.SnowFlakeEtcdHolder.EtcdAddressNode)
				uc.updateLocalWorkerID(uc.WorkerId)
			} else {
				// 不存在自己的节点则表示是一个新启动的节点，则创建持久节点，不需要check时间
				workId := 0

				// 找到最大的ID
				for _, id := range nodeMap {
					if workId < id {
						workId = id
					}
				}
				uc.SnowFlakeEtcdHolder.WorkerId = workId + 1
				uc.SnowFlakeEtcdHolder.EtcdAddressNode = PathForever + "/" + uc.ListenAddress +
					fmt.Sprintf("-%d", uc.SnowFlakeEtcdHolder.WorkerId)
				if success := uc.repo.CreateKeyWithOptLock(context.TODO(),
					uc.SnowFlakeEtcdHolder.EtcdAddressNode,
					string(uc.buildData())); !success {
					// 其他实例已创建
					if retryCount > 3 {
						return false
					}
					retryCount++
					goto RETRY
				}
				go uc.scheduledUploadData(uc.SnowFlakeEtcdHolder.EtcdAddressNode)
				uc.updateLocalWorkerID(uc.WorkerId)
			}
		}
	} else {
		uc.log.Error("start node ERROR : ", err)
		// 读不到etcd就从本地尝试读取
		if _, err := os.Stat(PropPath); err == nil {
			readFile, err := ioutil.ReadFile(PropPath)
			if err != nil {
				uc.log.Error("read file error : ", err)
				return false
			}
			split := strings.Split(string(readFile), "=")
			uc.WorkerId = cast.ToInt(split[1])
			uc.log.Warnf("START FAILED ,use local node file properties workerID-{%d}", uc.WorkerId)
		}
	}

	return true
}

func (uc *SegmentUsecase) updateLocalWorkerID(workId int) {
	if _, err := os.Stat(PropPath); err != nil {
		os.MkdirAll(filepath.Dir(PropPath), os.ModePerm)
	}

	err := ioutil.WriteFile(PropPath, []byte(fmt.Sprintf("workerID=%d", workId)), os.ModePerm)
	if err != nil {
		uc.log.Infof("%+v", err)
		return
	}
	return
}

func (uc *SegmentUsecase) scheduledUploadData(zkAddrNode string) {
	ticker := time.NewTicker(3 * time.Second)
	for {
		select {
		case <-ticker.C:
			uc.updateNewData(zkAddrNode)
		}
	}
}

func (uc *SegmentUsecase) updateNewData(path string) {
	if time.Now().UnixMilli() < uc.SnowFlakeEtcdHolder.LastUpdateTime {
		return
	}
	success := uc.repo.CreateOrUpdateKey(context.TODO(), path, string(uc.buildData()))
	if !success {
		return
	}
	uc.SnowFlakeEtcdHolder.LastUpdateTime = time.Now().UnixMilli()
	return
}

func (uc *SegmentUsecase) buildData() []byte {
	endPoint := new(model.Endpoint)
	endPoint.IP = uc.SnowFlakeEtcdHolder.Ip
	endPoint.Port = uc.SnowFlakeEtcdHolder.Port
	endPoint.Timestamp = time.Now().UnixMilli()
	encodeArr, _ := json.Marshal(endPoint)
	return encodeArr
}

func (uc *SegmentUsecase) deBuildData(val []byte) *model.Endpoint {
	endPoint := new(model.Endpoint)
	_ = json.Unmarshal(val, endPoint)
	return endPoint
}

func (uc *SegmentUsecase) checkInitTimeStamp(zkAddrNode string) bool {
	getKey, err := uc.repo.GetKey(context.TODO(), zkAddrNode)
	if err != nil {
		return false
	}
	endpoint := uc.deBuildData(getKey.Kvs[0].Value)
	return !(endpoint.Timestamp > time.Now().UnixMilli())
}
