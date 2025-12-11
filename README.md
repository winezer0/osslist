# 阿里云OSS文件列表树形展示工具

本工具用于列出阿里云OSS存储空间中的所有文件，并以树形结构展示。

## 功能特点

- 连接阿里云OSS并列出指定存储空间中的所有文件
- 以直观的树形结构展示文件和目录层次关系
- 支持指定前缀过滤，只显示特定目录下的文件
- 命令行界面，易于使用

## 安装和使用

### 前提条件

- Go 1.16 或更高版本
- 阿里云账号及OSS服务访问权限
- 拥有OSS的Access Key ID和Access Key Secret

### 安装依赖

```bash
go mod tidy
```

### 编译

```bash
go build -o osslist
```

### 使用方法

```bash
./osslist --accessKeyId=<accessKeyId> --accessKeySecret=<accessKeySecret> --bucket=<bucketName> [--endpoint=<endpoint>] [--prefix=<prefix>]
```

#### 参数说明

- `-e, --endpoint`: 阿里云OSS endpoint，例如：`http://oss-cn-hangzhou.aliyuncs.com`（可选）
- `-i, --accessKeyId`: 阿里云 Access Key ID（必需）
- `-s, --accessKeySecret`: 阿里云 Access Key Secret（必需）
- `-b, --bucket`: OSS存储空间名称（必需）
- `-p, --prefix`: 只列出具有指定前缀的文件，相当于只显示某个目录下的文件（可选）

#### 示例

```bash
# 列出存储空间中的所有文件（自动推断endpoint）
./osslist --accessKeyId=YOUR_ACCESS_KEY_ID --accessKeySecret=YOUR_ACCESS_KEY_SECRET --bucket=my-bucket

# 指定endpoint列出存储空间中的所有文件
./osslist --endpoint=http://oss-cn-hangzhou.aliyuncs.com --accessKeyId=YOUR_ACCESS_KEY_ID --accessKeySecret=YOUR_ACCESS_KEY_SECRET --bucket=my-bucket

# 只列出指定目录下的文件
./osslist --accessKeyId=YOUR_ACCESS_KEY_ID --accessKeySecret=YOUR_ACCESS_KEY_SECRET --bucket=my-bucket --prefix=documents/
```

### 直接运行（无需编译）

```bash
go run main.go --accessKeyId=<accessKeyId> --accessKeySecret=<accessKeySecret> --bucket=<bucketName>
```

## 输出示例

```
/
├── folder1/
│   ├── subfolder/
│   │   └── file2.txt
│   └── file1.txt
└── folder2/
    └── file3.txt
```

其中以 `/` 结尾的表示目录，否则为文件。

## 注意事项

1. 确保提供的Access Key拥有对指定Bucket的读取权限（至少需要`oss:ListObjects`权限）
2. 当不指定endpoint时，工具会尝试自动推断合适的endpoint
3. Endpoint格式请参考[阿里云OSS文档](https://help.aliyun.com/document_detail/31837.html)
4. 工具会加载存储空间中的所有文件信息，对于包含大量文件的存储空间，可能需要较长时间才能完成