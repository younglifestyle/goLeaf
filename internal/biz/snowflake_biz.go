package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/spf13/cast"
	clientv3 "go.etcd.io/etcd/client/v3"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"seg-server/internal/biz/model"
	"seg-server/internal/conf"
	"seg-server/internal/pkg/dir"
	"seg-server/internal/pkg/ip"
	mytime "seg-server/internal/pkg/time"
	"strings"
	"sync"
	"time"
)

var (
	// 起始的时间戳，用于用当前时间戳减去这个时间戳，算出偏移量
	twepoch            = int64(1288834974657)
	workerIdBits       = 10
	maxWorkerId        = ^(-1 << workerIdBits) // 1023
	sequenceBits       = 12
	workerIdShift      = sequenceBits
	timestampLeftShift = sequenceBits + workerIdBits
	sequenceMask       = int64(^(-1 << sequenceBits)) //4095 0xFFF
	PrefixEtcdPath     = "/snowflake/"
	PropPath           string
	PathForever        = PrefixEtcdPath + "/forever" //保存所有数据持久的节点
)

// SnowflakeIdGenUsecase is a snowflake usecase.
type SnowflakeIdGenUsecase struct {
	repo SnowflakeIDGenRepo
	conf *conf.Data

	twepoch       int64
	workerId      int64
	sequence      int64
	lastTimestamp int64
	snowFlakeLock sync.Mutex

	model.SnowFlakeEtcdHolder

	log *log.Helper
}

// SnowflakeIDGenRepo Etcd operate sets
type SnowflakeIDGenRepo interface {
	GetPrefixKey(ctx context.Context, prefix string) (*clientv3.GetResponse, error)
	CreateKeyWithOptLock(ctx context.Context, key string, val string) bool
	CreateOrUpdateKey(ctx context.Context, key string, val string) bool
	GetKey(ctx context.Context, key string) (*clientv3.GetResponse, error)
}

// NewSnowflakeIDGenUsecase new a snowflake usecase.
func NewSnowflakeIDGenUsecase(repo SnowflakeIDGenRepo, conf *conf.Bootstrap, logger log.Logger) *SnowflakeIdGenUsecase {
	s := &SnowflakeIdGenUsecase{
		repo: repo,
		conf: conf.Data,
		log:  log.NewHelper(log.With(logger, "module", "leaf-grpc/snowflake"))}

	if conf.Data.Etcd.SnowflakeEnable {
		initSnowflake(s, conf)
	}

	return s
}

// GetSnowflakeID creates a Snowflake ID  运行时，leaf允许最多5ms的回拨；重启时，允许最多3s的回拨
func (uc *SnowflakeIdGenUsecase) GetSnowflakeID(ctx context.Context) (int64, error) {
	if uc.conf.Etcd.SnowflakeEnable {
		// 多个共享变量会被并发访问，同一时间，应只让一个线程有资格访问数据
		uc.snowFlakeLock.Lock()
		defer uc.snowFlakeLock.Unlock()

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
				ts = mytime.UtilNextMillis(uc.lastTimestamp)
			}
		} else {
			uc.sequence = int64(rand.Intn(100))
		}
		uc.lastTimestamp = ts
		id := ((ts - uc.twepoch) << timestampLeftShift) | (uc.workerId << workerIdShift) | uc.sequence

		return id, nil
	} else {
		return 0, nil
	}
}

func initSnowflake(s *SnowflakeIdGenUsecase, conf *conf.Bootstrap) {
	s.twepoch = twepoch
	if !(time.Now().UnixMilli() > twepoch) {
		panic("Snowflake not support twepoch gt currentTime")
	}

	s.SnowFlakeEtcdHolder.Ip = ip.GetOutboundIP()
	s.SnowFlakeEtcdHolder.Port = strings.Split(conf.Server.Http.Addr, ":")[1]

	PrefixEtcdPath = "/snowflake/" + conf.Server.ServerName
	PropPath = filepath.Join(dir.GetCurrentAbPath(), conf.Server.ServerName) +
		"/leafconf/" + s.SnowFlakeEtcdHolder.Port + "/workerID.toml"
	PathForever = PrefixEtcdPath + "/forever"
	s.log.Info("workerID local cache file path : ", PropPath)

	s.SnowFlakeEtcdHolder.ListenAddress = s.SnowFlakeEtcdHolder.Ip + ":" + s.SnowFlakeEtcdHolder.Port
	if !s.initSnowflakeWorkId() {
		s.log.Error("Snowflake Service Init Fail")
		conf.Data.Etcd.SnowflakeEnable = false
	} else {
		s.workerId = int64(s.SnowFlakeEtcdHolder.WorkerId)
	}
	if !(s.workerId >= 0 && s.workerId <= int64(maxWorkerId)) {
		panic("workerID must gte 0 and lte 1023")
	}
}

func (uc *SnowflakeIdGenUsecase) initSnowflakeWorkId() bool {
	var retryCount = 0
RETRY:
	prefixKeyResps, err := uc.repo.GetPrefixKey(newTimeoutCtx(time.Second*2), PathForever)
	if err == nil {
		// 还没有实例化过
		if prefixKeyResps.Count == 0 {
			uc.SnowFlakeEtcdHolder.EtcdAddressNode = PathForever + "/" + uc.ListenAddress + "-0"
			if success := uc.repo.CreateKeyWithOptLock(newTimeoutCtx(time.Second),
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
				if success := uc.repo.CreateKeyWithOptLock(newTimeoutCtx(time.Second),
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
		} else {
			uc.log.Error("workerID file not exist...")
			return false
		}
	}

	return true
}

func (uc *SnowflakeIdGenUsecase) updateLocalWorkerID(workId int) {
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

func (uc *SnowflakeIdGenUsecase) scheduledUploadData(zkAddrNode string) {
	ticker := time.NewTicker(3 * time.Second)
	for {
		select {
		case <-ticker.C:
			uc.updateNewData(zkAddrNode)
		}
	}
}

func (uc *SnowflakeIdGenUsecase) updateNewData(path string) {
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

func (uc *SnowflakeIdGenUsecase) buildData() []byte {
	endPoint := new(model.Endpoint)
	endPoint.IP = uc.SnowFlakeEtcdHolder.Ip
	endPoint.Port = uc.SnowFlakeEtcdHolder.Port
	endPoint.Timestamp = time.Now().UnixMilli()
	encodeArr, _ := json.Marshal(endPoint)
	return encodeArr
}

func (uc *SnowflakeIdGenUsecase) deBuildData(val []byte) *model.Endpoint {
	endPoint := new(model.Endpoint)
	_ = json.Unmarshal(val, endPoint)
	return endPoint
}

func (uc *SnowflakeIdGenUsecase) checkInitTimeStamp(zkAddrNode string) bool {
	getKey, err := uc.repo.GetKey(context.TODO(), zkAddrNode)
	if err != nil {
		return false
	}
	endpoint := uc.deBuildData(getKey.Kvs[0].Value)
	return !(endpoint.Timestamp > time.Now().UnixMilli())
}

func newTimeoutCtx(duration time.Duration) context.Context {
	// timeout ctx会自动超时，cancel不需要调用
	timeoutCtx, _ := context.WithTimeout(context.TODO(), duration)
	return timeoutCtx
}
