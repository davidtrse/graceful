package log

import (
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

type ZapConfig struct {
	ZapEnvironment    string `mapstructure:"zap_environment"`
	ZapEncodeMethod   string `mapstructure:"zap_encode_method"`
	ZapFilePath       string `mapstructure:"zap_file_path"`
	ZapMaxSizeMb      int    `mapstructure:"zap_max_size_mb"`
	ZapMaxBackupFiles int    `mapstructure:"zap_max_backup_files"`
	ZapMaxAgeDays     int    `mapstructure:"zap_max_age_days"`
}

var DefaultZapConfig = &ZapConfig{
	ZapEnvironment:    "DEV",
	ZapEncodeMethod:   "console",
	ZapFilePath:       "/tmp/vod.log",
	ZapMaxSizeMb:      10,
	ZapMaxBackupFiles: 10,
	ZapMaxAgeDays:     10,
}

func GetEncoderConfig(zc *ZapConfig) zapcore.EncoderConfig {
	var encoderConfig zapcore.EncoderConfig
	if zc.ZapEnvironment == "PROD" {
		encoderConfig = zap.NewProductionEncoderConfig()
	} else {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
	}
	// encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	// encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeTime = SyslogTimeEncoder
	encoderConfig.EncodeLevel = CustomLevelEncoder
	encoderConfig.StacktraceKey = ""
	encoderConfig.CallerKey = "caller"
	encoderConfig.LevelKey = "level"
	encoderConfig.MessageKey = "message"
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	return encoderConfig
}

func GetLogWriter(zc *ZapConfig) zapcore.WriteSyncer {
	return zapcore.AddSync(&lumberjack.Logger{
		Filename:   zc.ZapFilePath,
		MaxSize:    zc.ZapMaxSizeMb,
		MaxBackups: zc.ZapMaxBackupFiles,
		MaxAge:     zc.ZapMaxAgeDays,
	})
}

func SyslogTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000Z"))
}

func CustomLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + level.CapitalString() + "]")
}
