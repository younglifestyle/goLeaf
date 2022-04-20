## go-leaf 

### 介绍
Go实现的号段模式发号器，基于Kratos框架，适用于此微服务框架以及服务发现服务

gRPC访问性能与Leaf同

### 使用

- 创建表
```mysql
CREATE DATABASE leaf;
CREATE TABLE `leaf_alloc` (
    `biz_tag` varchar(128)  NOT NULL DEFAULT '', -- your biz unique name
    `max_id` bigint(20) NOT NULL DEFAULT '1',
    `step` int(11) NOT NULL,
    `description` varchar(256)  DEFAULT NULL,
    `update_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `created_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`biz_tag`)
) ENGINE=InnoDB;

insert into leaf_alloc(biz_tag, max_id, step, description) values('leaf-segment-test', 1, 2000, 'Test leaf Segment Mode Get Id');
```

- 启动服务
```
make build
bin/seq-server -conf configs/config.yaml
```

- 请求接口
```
curl http://localhost:8000/api/segment/get/leaf-segment-test
```