# Leaf

> There are no two identical leaves in the world.
>
> ​               — Leibnitz
> 
[中文文档](./README_zh.md) | [English Document](./README.md)

## Introduce


A number segment pattern and Snowflake ID generator implemented in Go, based on the Kratos framework, suitable for this microservices framework and service discovery service.

The gRPC access performance is on par with Leaf.

> Add the functionality to automatically reset the serial number every day (add the `auto_clean` field, intended for single-node use, with Nginx backup).
> 
> Add an api to support remote addition of number segment tags.
> 
> Add an api for bulk retrieval of number segments.
> 
> Introduce a new UI interface that supports adding, viewing number segments, caching, and parsing Snowflake IDs.

## Quick Start

### Segment Pattern

- table

```mysql
CREATE DATABASE leaf;

CREATE TABLE `leaf_alloc` (
    `biz_tag` varchar(128)  NOT NULL DEFAULT '', -- your biz unique name
    `max_id` bigint(20) NOT NULL DEFAULT '1',
    `step` int(11) NOT NULL,
    `description` varchar(256)  DEFAULT NULL,
    `update_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `created_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `auto_clean` tinyint(1) default 0 null,
    PRIMARY KEY (`biz_tag`)
) ENGINE=InnoDB;

insert into leaf_alloc(biz_tag, max_id, step, description) values('leaf-segment-test', 1, 2000, 'Test leaf Segment Mode Get Id');
```

- config

```yaml
data:
  database:
    segment_enable: true
    table_name: "leaf_alloc"
    driver: mysql
    source: root:123@tcp(127.0.0.1:3306)/leaf?charset=utf8&parseTime=True&loc=Local
    open_conn: 50
    idle: 10
    idle_timeout: 14400s
```

- run

```
make build
bin/seq-server -conf configs/config.yaml
```

- api

```
curl http://localhost:8000/api/segment/get/leaf-segment-test

// cache data
http://localhost:8000/monitor/cache
// db segment data
http://localhost:8000/monitor/db
```
### Snowflake Pattern

- config

```yaml
data:
  etcd:
    snowflake_enable: true
    # Enabling the functionality to synchronize and validate time with other Leaf nodes.
    discovery_enable: false
    endpoints: ["127.0.0.1:2379"]
    dial_timeout: 2s
    # When `discovery_enable` is enabled, the deviation in time among service nodes.
    time_deviation: 50
```

- api

```
curl http://localhost:8000/api/snowflake/get

// decode ID
http://localhost:8000/decodeSnowflakeId
```

### UI

addr：http:port/web

> 例如：http://127.0.0.1:8001/web

![DB号段](doc/image-20221101201529905.png)

---

![cache中的号段](doc/image-20221101201630411.png)

---

![雪花ID解析](doc/image-20221101201705230.png)
