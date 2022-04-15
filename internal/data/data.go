package data

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"go.etcd.io/etcd/client/v3"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"seg-server/internal/conf"
	"sync"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewGormClient, NewSegmentIDRepo)

// Data .
type Data struct {
	db        *gorm.DB
	tableName string
	RWMutex   *sync.RWMutex
	log       *log.Helper
}

// NewData .
func NewData(conf *conf.Data, db *gorm.DB, logger log.Logger) (*Data, func(), error) {
	l := log.NewHelper(log.With(logger, "module", "leaf-grpc/data"))

	cleanup := func() {
		l.Info("closing the data resources")
		s, err := db.DB()
		if err == nil {
			s.Close()
		}
	}
	return &Data{db: db, RWMutex: &sync.RWMutex{},
		tableName: conf.Database.TableName, log: l}, cleanup, nil
}

// NewEtcdClient 创建 Etcd 客户端
func NewEtcdClient(conf *conf.Data, logger log.Logger) *clientv3.Client {
	l := log.NewHelper(log.With(logger, "module", "redis/data/logger-job"))

	c, err := clientv3.New(clientv3.Config{
		Endpoints:   conf.Etcd.Endpoints,
		DialTimeout: conf.Etcd.DialTimeout.AsDuration(),
	})
	if err != nil {
		l.Fatalf("failed opening connection to etcd")
	}

	return c
}

// NewGormClient 创建数据库客户端
func NewGormClient(c *conf.Data, logger log.Logger) *gorm.DB {
	l := log.NewHelper(log.With(logger, "module", "gorm/data"))

	// 参考 https://github.com/go-sql-driver/mysql#dsn-data-source-name 获取详情
	db, err := gorm.Open(mysql.Open(c.Database.Source), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	})
	if err != nil {
		l.Fatalf("failed opening connection to db: %v", err)
	}

	// 获取通用数据库对象 sql.DB ，然后使用其提供的功能
	sqlDB, err := db.DB()
	if err != nil {
		l.Fatalf("failed get connection for db: %v", err)
	}

	// SetMaxIdleConns 用于设置连接池中空闲连接的最大数量。
	sqlDB.SetMaxIdleConns(int(c.Database.Idle))
	sqlDB.SetConnMaxIdleTime(c.Database.IdleTimeout.AsDuration())
	// SetMaxOpenConns 设置打开数据库连接的最大数量。
	sqlDB.SetMaxOpenConns(int(c.Database.OpenConn))
	return db
}
