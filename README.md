# 阿里云OSS文件展示工具

本工具用于列出阿里云OSS存储空间中的所有文件。

## 功能特点

- 连接阿里云OSS并列出指定存储空间中的所有文件
- 支持指定 前缀/路径/后缀 过滤，只显示特定目录下的文件
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

## 注意事项

1. 确保提供的Access Key拥有对指定Bucket的读取权限（至少需要`oss:ListObjects`权限）
2. 当不指定endpoint时，工具会尝试自动推断合适的endpoint
3. Endpoint格式请参考[阿里云OSS文档](https://help.aliyun.com/document_detail/31837.html)
4. 工具会加载存储空间中的所有文件信息，对于包含大量文件的存储空间，可能需要较长时间才能完成