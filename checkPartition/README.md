# checkPartition - GoldenDB 分区一致性检查工具

`checkPartition` 是一个用于检查 GoldenDB 集群中分区表在各个数据节点（DN）上的分区信息是否一致的工具。它能够自动发现集群、Schema 和分区表，并对比所有副本的分区名称和顺序。

## 功能特性

- **自动发现**：连接 MDS 自动获取集群拓扑、Schema 和 DN 节点信息。
- **分区一致性检查**：对比所有 DN 节点上同一张表的分区名称和顺序。
- **报告生成**：发现不一致时，生成 JSON 格式的错误报告（`YYYYMMDD-ddlerror.json`）。
- **密码加密**：提供命令行工具加密数据库密码，确保配置文件安全。
- **多平台支持**：支持 Linux (amd64, arm) 和 macOS (arm64)。

## 依赖环境

- Go 1.18+ (如果需要从源码编译)
- 网络能够访问 MDS 和 DN 节点

## 编译

使用提供的构建脚本进行跨平台编译：

```bash
./build.sh
```

编译完成后，会在当前目录下生成以下二进制文件：
- `checkpartition-linux-amd64`
- `checkpartition-linux-arm`
- `checkpartition-darwin-arm64`

或者直接使用 `go build` 编译当前平台的版本：

```bash
go build -o checkpartition main.go
```

## 配置

工具默认读取 `config/mds.json` 文件作为 MDS 连接配置。请确保该文件存在并配置正确。

**配置文件路径**：`config/mds.json`

**格式示例**：

```json
[
  {
    "name" : "cluster1_mds",
    "username": "root",
    "password": "ENCRYPTED_PASSWORD_HERE",
    "host": "192.168.1.100",
    "port": 3309
  },
  {
    "name" : "cluster2_mds",
    "username": "root",
    "password": "ENCRYPTED_PASSWORD_HERE",
    "host": "192.168.1.101",
    "port": 3309
  }
]
```

> **注意**：`password` 字段必须是经过工具加密后的密文，不能使用明文。

## 使用方法

### 1. 生成加密密码

在使用配置文件之前，需要先将明文密码加密。

```bash
./checkpartition -p <明文密码>
```

示例：
```bash
$ ./checkpartition -p 123456
i/HfDbXu3+zCAq1YwiJgZkrlEi0tO1//8ho/tu5z7/DdrWwcIg==
```

将生成的密文填入 `config/mds.json` 的 `password` 字段。

### 2. 执行检查

配置完成后，使用 `-s` 参数启动检查：

```bash
./checkpartition -s
```

工具会连接配置的 MDS，遍历所有集群和 Schema，检查分区表的一致性。

### 3. 查看结果

- **控制台输出**：实时显示检查进度和发现的问题。
  - `分区表 {Schema TableName} 分区信息一致`
  - `分区表 {Schema TableName} 分区信息不一致`
- **错误报告**：如果发现不一致，会在运行目录下生成 `YYYYMMDD-ddlerror.json` 文件。

**错误报告示例**：

```json
[
  {
    "Cluster": "cluster_name",
    "Schema": "db_name",
    "Table": "table_name",
    "Error": "租户cluster_name的db_name库的table_name分区信息不一致"
  }
]
```

## 目录结构

- `alarm/`: 集群拓扑信息获取逻辑
- `check/`: 分区信息检查核心逻辑
- `config/`: 配置文件
- `connect/`: 数据库连接与加解密工具
- `main.go`: 程序入口
- `build.sh`: 构建脚本
