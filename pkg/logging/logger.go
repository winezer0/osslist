package logging

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// -------------------------- 跨包全局日志器(按需定义) --------------------------
// 定义需要跨包使用的全局日志器，建议用明确的语义化命名(避免泛称如"newLogger")
//var (
//	// APILogger API模块专用日志器(供所有包调用)
//	APILogger *Logger
// 第一步：优先初始化全局日志器(必须在所有业务逻辑前) 程序退出时关闭所有日志器，刷新缓冲区
//if err := log.InitGlobalLoggers(); err != nil { panic(fmt.Sprintf("init global loggers failed: %+v", err)) }
//defer log.CloseAll()

// -------------------------- 配置定义 --------------------------

// LogConfig 日志配置结构体，同时支持新老版本使用
type LogConfig struct {
	Level         string // 日志级别: debug/info/warn/error/fatal
	LogFile       string // 日志文件路径，空串表示不输出到文件
	ConsoleFormat string // 控制台格式: 空串或"off"表示关闭，支持"T(时间)L(级别)C(调用者)M(消息)"
}

// NewLogConfig 创建日志配置实例，提供默认值
func NewLogConfig(level, logFile, consoleFormat string) LogConfig {
	if level == "" {
		level = "info" // 默认info级别
	}
	return LogConfig{
		Level:         level,
		LogFile:       logFile,
		ConsoleFormat: consoleFormat,
	}
}

// -------------------------- 日志器实现 --------------------------

// Logger 日志器实例，线程安全
type Logger struct {
	zapLogger *zap.Logger
	sugar     *zap.SugaredLogger
	config    LogConfig
	mu        sync.RWMutex
}

// init 初始化日志器核心(已修正EncodeTime配置)
func (l *Logger) init() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 解析日志级别
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(l.config.Level)); err != nil {
		level = zapcore.InfoLevel // 默认为info级别
	}

	// 准备输出核心
	var cores []zapcore.Core

	// 控制台输出
	if l.config.ConsoleFormat != "" && l.config.ConsoleFormat != "off" {
		encoder := newConsoleEncoder(l.config.ConsoleFormat)
		cores = append(cores, zapcore.NewCore(
			encoder,
			zapcore.Lock(os.Stdout),
			level,
		))
	}

	// 文件输出(带日志轮转，正确配置时间格式)
	if l.config.LogFile != "" {
		if err := EnsureDir(l.config.LogFile); err != nil {
			return fmt.Errorf("创建日志目录失败: %w", err)
		}

		// 日志轮转配置
		rotator := &lumberjack.Logger{
			Filename:   l.config.LogFile,
			MaxSize:    10,   // 单个文件最大100MB
			MaxBackups: 10,   // 最多保留10个备份
			MaxAge:     10,   // 保留30天
			Compress:   true, // 压缩备份文件
		}

		// 配置文件日志编码器(含时间格式)
		fileEncoderCfg := zap.NewProductionEncoderConfig()
		// 配置时间格式为ISO8601(如：2024-05-20T15:30:00.000Z)
		fileEncoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
		// (可选)自定义时间格式示例(如：2024-05-20 15:30:00.000)
		// fileEncoderCfg.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		// 	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
		// }

		// 创建JSON格式编码器
		fileEncoder := zapcore.NewJSONEncoder(fileEncoderCfg)
		cores = append(cores, zapcore.NewCore(
			fileEncoder,
			zapcore.AddSync(rotator),
			level,
		))
	}

	if len(cores) == 0 {
		return fmt.Errorf("未配置日志输出目标(控制台/文件)")
	}

	// 创建zap日志器
	l.zapLogger = zap.New(
		zapcore.NewTee(cores...),
		zap.AddCaller(),      // 显示调用位置(如 main.go:20)
		zap.AddCallerSkip(2), // 跳过内部方法，显示真实业务代码位置
	)
	l.sugar = l.zapLogger.Sugar()

	return nil
}

// 日志输出方法
func (l *Logger) Debugf(template string, args ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.sugar != nil {
		l.sugar.Debugf(template, args...)
	}
}

func (l *Logger) Infof(template string, args ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.sugar != nil {
		l.sugar.Infof(template, args...)
	}
}

func (l *Logger) Warnf(template string, args ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.sugar != nil {
		l.sugar.Warnf(template, args...)
	}
}

func (l *Logger) Errorf(template string, args ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.sugar != nil {
		l.sugar.Errorf(template, args...)
	}
}

func (l *Logger) Fatalf(template string, args ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.sugar != nil {
		l.sugar.Fatalf(template, args...)
	}
}

func (l *Logger) Debug(args ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.sugar != nil {
		l.sugar.Debug(args...)
	}
}

func (l *Logger) Info(args ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.sugar != nil {
		l.sugar.Info(args...)
	}
}

func (l *Logger) Warn(args ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.sugar != nil {
		l.sugar.Warn(args...)
	}
}

func (l *Logger) Error(args ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.sugar != nil {
		l.sugar.Error(args...)
	}
}

func (l *Logger) Fatal(args ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.sugar != nil {
		l.sugar.Fatal(args...)
	}
}

// Sync 刷新日志缓冲区
func (l *Logger) Sync() error {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.zapLogger != nil {
		return l.zapLogger.Sync()
	}
	return nil
}

// -------------------------- 日志器管理器 --------------------------

type loggerManager struct {
	loggers map[string]*Logger
	mu      sync.RWMutex
}

var (
	globalManager *loggerManager
	once          sync.Once
)

func getManager() *loggerManager {
	once.Do(func() {
		globalManager = &loggerManager{
			loggers: make(map[string]*Logger),
		}
	})
	return globalManager
}

// CreateLogger 创建新的日志器
func CreateLogger(name string, config LogConfig) (*Logger, error) {
	if name == "" {
		return nil, fmt.Errorf("日志器名称不能为空")
	}

	manager := getManager()
	manager.mu.RLock()
	_, exists := manager.loggers[name]
	manager.mu.RUnlock()
	if exists {
		return nil, fmt.Errorf("日志器 '%s' 已存在", name)
	}

	logger := &Logger{config: config}
	if err := logger.init(); err != nil {
		return nil, err
	}

	manager.mu.Lock()
	manager.loggers[name] = logger
	manager.mu.Unlock()

	return logger, nil
}

// GetLogger 获取已创建的日志器
func GetLogger(name string) (*Logger, bool) {
	manager := getManager()
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	logger, exists := manager.loggers[name]
	return logger, exists
}

// CloseAll 关闭所有日志器
func CloseAll() error {
	manager := getManager()
	manager.mu.Lock()
	defer manager.mu.Unlock()

	var errList []error
	for name, logger := range manager.loggers {
		if err := logger.Sync(); err != nil {
			errList = append(errList, fmt.Errorf("关闭日志器 '%s' 失败: %w", name, err))
		}
	}
	manager.loggers = make(map[string]*Logger)
	return errors.Join(errList...)
}

// -------------------------- 旧版本兼容层 --------------------------

var defaultLogger *Logger // 旧版本默认日志器

// InitLogger 旧版本初始化函数，兼容老代码
func InitLogger(config LogConfig) error {
	var err error
	defaultLogger, err = CreateLogger("default", config)
	return err
}

// 旧版本全局日志函数，直接转发到default日志器
func Debugf(template string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Debugf(template, args...)
	}
}

func Infof(template string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Infof(template, args...)
	}
}

func Warnf(template string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Warnf(template, args...)
	}
}

func Errorf(template string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Errorf(template, args...)
	}
}

func Fatalf(template string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Fatalf(template, args...)
	}
}

func Debug(args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Debug(args...)
	}
}

func Info(args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Info(args...)
	}
}

func Warn(args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Warn(args...)
	}
}

func Error(args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Error(args...)
	}
}

func Fatal(args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Fatal(args...)
	}
}

// Sync 旧版本全局刷新函数
func Sync() error {
	if defaultLogger != nil {
		return defaultLogger.Sync()
	}
	return nil
}

// -------------------------- 工具函数 --------------------------

// 创建控制台编码器
func newConsoleEncoder(format string) zapcore.Encoder {
	cfg := zapcore.EncoderConfig{
		TimeKey:      "T",
		LevelKey:     "L",
		CallerKey:    "C",
		MessageKey:   "M",
		EncodeTime:   zapcore.ISO8601TimeEncoder,
		EncodeLevel:  zapcore.CapitalLevelEncoder,
		EncodeCaller: zapcore.ShortCallerEncoder,
	}

	if !strings.Contains(format, "T") {
		cfg.TimeKey = ""
	}
	if !strings.Contains(format, "L") {
		cfg.LevelKey = ""
	}
	if !strings.Contains(format, "C") {
		cfg.CallerKey = ""
	}
	if !strings.Contains(format, "M") {
		cfg.MessageKey = ""
	}

	return zapcore.NewConsoleEncoder(cfg)
}

// 确保目录存在
func EnsureDir(filePath string) error {
	dir := filepath.Dir(filePath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}
