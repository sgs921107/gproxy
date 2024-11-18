package glogging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
	"strings"
	"sync"
)

type UTCEncoder struct {
	zapcore.Encoder
}

func (e *UTCEncoder) EncodeEntry(ent zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	ent.Time = ent.Time.UTC()
	return e.Encoder.EncodeEntry(ent, fields)
}

// 起别名
type (
	// ZapLogger zap logger
	ZapLogger = zap.Logger
	// ZapSugaredLogger zap sugared logger
	ZapSugaredLogger = zap.SugaredLogger
)

// ZapLogging	zap logging
type ZapLogging interface {
	BaseLogging
	GetLogger() *ZapLogger
	GetSugaredLogger() *ZapSugaredLogger
	Sync()
}

// ZapLog	zap log
type zapLog struct {
	baseLog
	once          sync.Once
	logger        *ZapLogger
	sugaredOnce   sync.Once
	sugaredLogger *ZapSugaredLogger
}

// GetLogger  获取一个zap的logger
func (l *zapLog) GetLogger() *ZapLogger {
	l.once.Do(func() {
		zapCore := zapcore.NewCore(l.encoder(), l.writer(), l.level())
		l.logger = zap.New(zapCore, zap.AddCaller())
		zap.ReplaceGlobals(l.logger)
	})
	return l.logger
}

// GetSugaredLogger 获取一个zap的sugared logger
func (l *zapLog) GetSugaredLogger() *ZapSugaredLogger {
	l.sugaredOnce.Do(func() {
		logger := l.GetLogger()
		l.sugaredLogger = logger.Sugar()
	})
	return l.sugaredLogger
}

// create zap log level
func (l *zapLog) level() zapcore.LevelEnabler {
	switch strings.ToUpper(l.options.Level) {
	case "DEBUG":
		return zapcore.DebugLevel
	case "INFO":
		return zapcore.InfoLevel
	case "WARNING":
		return zapcore.WarnLevel
	case "ERROR":
		return zapcore.ErrorLevel
	case "DPANIC":
		return zapcore.DPanicLevel
	case "PANIC":
		return zapcore.PanicLevel
	case "FATAL":
		return zapcore.FatalLevel
	default:
		return zapcore.DebugLevel
	}

}

// create zap encoder
func (l *zapLog) encoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	// 格式时间
	encoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(l.options.TimeFormater)
	encoderConfig.CallerKey = "file"
	encoderConfig.TimeKey = "time"
	// 大写形式显示日志等级
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	// 默认展示简短的调用者信息
	if l.options.Caller == "full" {
		encoderConfig.EncodeCaller = zapcore.FullCallerEncoder
	} else {
		encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	}
	var encoder zapcore.Encoder
	switch strings.ToUpper(l.options.Formatter) {
	case "TEXT":
		// 文本格式
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	default:
		// json格式
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}
	if l.options.UseUTC {
		return &UTCEncoder{encoder}
	} else {
		return encoder
	}
}

// create zap writer
func (l *zapLog) writer() zapcore.WriteSyncer {
	return zapcore.AddSync(l.Output())
}

// Sync	刷新内容
func (l *zapLog) Sync() {
	l.logger.Sync()
}

// NewZapLogging	new a zap logging
func NewZapLogging(options Options) ZapLogging {
	return &zapLog{
		baseLog: baseLog{
			options: options,
		},
	}
}
