package scacher

import (
	"encoding/json"
	"os"
	"sync"
)

// Cacher 表示一个简单的缓存管理器
type Cacher struct {
	cache       map[string]string
	filePath    string
	mutex       sync.RWMutex
	initialized bool
}

// NewCacher 创建一个新的Cacher实例
func NewCacher(filePath string) *Cacher {
	c := &Cacher{
		cache:       make(map[string]string),
		filePath:    filePath,
		initialized: false,
	}
	// 初始化时加载缓存文件
	c.loadCache()
	return c
}

// loadCache 从文件加载缓存数据
func (c *Cacher) loadCache() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// 检查文件是否存在
	if _, err := os.Stat(c.filePath); os.IsNotExist(err) {
		c.initialized = true
		return
	}

	// 读取文件内容
	data, err := os.ReadFile(c.filePath)
	if err != nil {
		c.initialized = true
		return
	}

	// 解析JSON数据
	if err := json.Unmarshal(data, &c.cache); err != nil {
		c.cache = make(map[string]string)
	}

	c.initialized = true
}

// saveCache 将缓存数据保存到文件
func (c *Cacher) saveCache() {
	c.mutex.RLock()
	// 检查是否初始化完成
	if !c.initialized {
		c.mutex.RUnlock()
		return
	}

	// 序列化缓存数据
	data, err := json.MarshalIndent(c.cache, "", "  ")
	c.mutex.RUnlock()

	if err != nil {
		return
	}

	// 写入文件
	if err := os.WriteFile(c.filePath, data, 0644); err != nil {
		return
	}
}

// Sync 同步缓存到文件
func (c *Cacher) Sync() {
	c.saveCache()
}

// Get 查询缓存，根据key获取value
func (c *Cacher) Get(key string) (string, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	value, exists := c.cache[key]
	return value, exists
}

// Set 更新缓存，设置key-value对
func (c *Cacher) Set(key, data string) {
	c.mutex.Lock()
	c.cache[key] = data
	c.mutex.Unlock()

	// 异步保存到文件
	go c.saveCache()
}

// Delete 移除指定key的缓存
func (c *Cacher) Delete(key string) {
	c.mutex.Lock()
	delete(c.cache, key)
	c.mutex.Unlock()

	// 异步保存到文件
	go c.saveCache()
}

// Clear 清空所有缓存并删除缓存文件
func (c *Cacher) Clear() error {
	c.mutex.Lock()
	c.cache = make(map[string]string)
	c.mutex.Unlock()

	// 异步保存到文件
	go c.saveCache()

	// 删除缓存文件
	if _, err := os.Stat(c.filePath); err == nil {
		if err := os.Remove(c.filePath); err != nil {
			return err
		}
	}

	return nil
}
