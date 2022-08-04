package data

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"seg-server/internal/conf"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewGormClient, NewEtcdClient, NewSegmentIdGenRepo, NewSnowflakeRepo)

// Data .
type Data struct {
	db        *gorm.DB
	etcdCli   *clientv3.Client
	tableName string
	log       *log.Helper
}

// NewData .
func NewData(conf *conf.Data, db *gorm.DB, cli *clientv3.Client, logger log.Logger) (*Data, func(), error) {
	l := log.NewHelper(log.With(logger, "module", "leaf-grpc-repo/data"))

	cleanup := func() {
		l.Info("closing the data resources")
		if conf.Database.SegmentEnable {
			s, err := db.DB()
			if err == nil {
				s.Close()
			}
		}
		if conf.Etcd.SnowflakeEnable {
			cli.Close()
		}
	}
	return &Data{db: db, etcdCli: cli, tableName: conf.Database.TableName, log: l}, cleanup, nil
}

// NewEtcdClient 创建 Etcd 客户端
func NewEtcdClient(c *conf.Data, logger log.Logger) (cli *clientv3.Client) {
	if c.Etcd.SnowflakeEnable || c.Etcd.DiscoveryEnable {
		var err error
		l := log.NewHelper(log.With(logger, "module", "leaf-grpc-repo/etcd_cli"))

		cli, err = clientv3.New(clientv3.Config{
			Endpoints:   c.Etcd.Endpoints,
			DialTimeout: c.Etcd.DialTimeout.AsDuration(),
			DialOptions: []grpc.DialOption{grpc.WithBlock()},
		})
		if err != nil {
			l.Errorf("failed opening connection to etcd : %s", err)
		}
	}

	//r := etcd.New(cli)
	//r.GetService(context.Background())

	return cli
}

// NewGormClient 创建数据库客户端
func NewGormClient(c *conf.Data, logger log.Logger) (db *gorm.DB) {
	if c.Database.SegmentEnable {
		var err error
		l := log.NewHelper(log.With(logger, "module", "leaf-grpc-repo/gorm"))

		// 参考 https://github.com/go-sql-driver/mysql#dsn-data-source-name 获取详情
		db, err = gorm.Open(mysql.Open(c.Database.Source), &gorm.Config{
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
	}
	return db
}
