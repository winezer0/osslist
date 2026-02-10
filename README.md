# 阿里云OSS文件展示工具

本工具用于列出阿里云OSS存储空间中的所有文件。


## 免责声明

继续阅读文章或使用工具视为您已同意《 NOVASEC免责声明》: [NOVASEC免责声明](https://mp.weixin.qq.com/s/iRWRVxkYu7Fx5unxA34I7g)



## 功能特点

- 连接阿里云OSS并列出指定存储空间中的所有文件
- 支持指定Bucket名称，可选择列出所有Bucket或特定Bucket
- 支持指定自定义Endpoint
- 支持指定前缀过滤，只显示特定目录下的文件
- 支持排除特定文件扩展名（如mp4,jpg等） 默认排除扩展名列表（mp3, woff, woff2, css, mp4, jpg, png, avi, mov等）
- 支持排除特定目录关键字（如temp,cache等） 默认排除目录关键字列表（chunks, temp, cache）
- 支持自定义并发工作线程数，提高列举速度

## 优点

- 高性能：支持并发列举，可根据需要调整工作线程数
- 灵活性：提供丰富的过滤选项，可根据实际需求定制列举范围
- 可配置性：支持详细的日志配置，便于问题排查

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

#### 命令行参数

| 参数 | 简写 | 说明 | 默认值 | 必填 |
|------|------|------|--------|------|
| --accessKeyId | -i | 阿里云Access Key ID | 无 | 是 |
| --accessKeySecret | -k | 阿里云Access Key Secret | 无 | 是 |
| --bucket | -b | Bucket名称，不指定则列出所有Bucket | 无 | 否 |
| --endpoint | -e | 阿里云OSS Endpoint | https://oss.aliyuncs.com | 否 |
| --prefix | -p | 只列出指定前缀的文件 | 无 | 否 |
| --et | 无 | 排除指定的文件扩展名，例如：mp4,jpg | 无 | 否 |
| --ET | 无 | 使用默认排除扩展名列表 | false | 否 |
| --ek | 无 | 排除指定的目录关键字，例如：temp,cache | 无 | 否 |
| --EK | 无 | 使用默认排除目录关键字列表 | false | 否 |
| --output | -o | 输出结果到文件 | osslist.txt | 否 |
| --workers | -w | 每个Bucket的并发工作线程数 | 10 | 否 |

#### 使用示例

1. **列出所有Bucket中的文件**
   ```bash
   osslist -i <accessKeyId> -k <accessKeySecret>
   ```

2. **列出指定Bucket中的文件**
   ```bash
   osslist -i <accessKeyId> -k <accessKeySecret> -b <bucketName>
   ```

3. **列出指定Bucket中特定前缀的文件**
   ```bash
   osslist -i <accessKeyId> -k <accessKeySecret> -b <bucketName> -p <prefix>
   ```

4. **使用自定义Endpoint**
   ```bash
   osslist -i <accessKeyId> -k <accessKeySecret> -b <bucketName> -e <endpoint>
   ```

5. **排除特定文件扩展名**
   ```bash
   osslist -i <accessKeyId> -k <accessKeySecret> -b <bucketName> --et mp4,jpg,png
   ```

6. **使用默认排除扩展名列表**
   ```bash
   osslist -i <accessKeyId> -k <accessKeySecret> -b <bucketName> --ET
   ```

7. **排除特定目录关键字**
   ```bash
   osslist -i <accessKeyId> -k <accessKeySecret> -b <bucketName> --ek temp,cache
   ```

8. **使用默认排除目录关键字列表**
   ```bash
   osslist -i <accessKeyId> -k <accessKeySecret> -b <bucketName> --EK
   ```

9. **指定输出文件**
   ```bash
   osslist -i <accessKeyId> -k <accessKeySecret> -b <bucketName> -o <outputFile>
   ```

10. **调整并发工作线程数**
    ```bash
    osslist -i <accessKeyId> -k <accessKeySecret> -b <bucketName> -w 20
    ```


### 注意事项

1. 确保提供的Access Key拥有对指定Bucket的读取权限（至少需要`oss:ListObjects`权限）
2. 当不指定endpoint时，工具会尝试自动推断合适的endpoint
3. Endpoint格式请参考[阿里云OSS文档](https://help.aliyun.com/document_detail/31837.html)
4. 工具会加载存储空间中的所有文件信息，对于包含大量文件的存储空间，可能需要较长时间才能完成