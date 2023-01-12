package log

import (
	"context"
	"fmt"
	"log"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

var (
	logger      *zap.Logger
	sugarLogger *zap.SugaredLogger
)

var CLIENT_ERROR = "CLIENT_ERROR"
var SERVER_ERROR = "SERVER_ERROR"

func init() {
	Init(DefaultZapConfig)
}

func Init(zc *ZapConfig) {
	fmt.Println("Init Zap logger")

	var err error
	zapConfig := CreateZapConfig(zc)

	logger, err = zapConfig.Build(zap.AddCallerSkip(1))
	if err != nil {
		log.Fatal("Creating Zap Logger error.")
	}
	sugarLogger = logger.Sugar()

	defer func() {
		_ = logger.Sync()
		_ = sugarLogger.Sync()
	}()
}

func CreateZapConfig(zc *ZapConfig) *zap.Config {
	var zapConfig zap.Config

	if zc.ZapEnvironment == "PROD" {
		zapConfig = zap.NewProductionConfig()
	} else {
		zapConfig = zap.NewDevelopmentConfig()
	}

	//Setting encode default is console
	zapConfig.EncoderConfig = GetEncoderConfig(zc)
	if zc.ZapEncodeMethod == "console" {
		zapConfig.Encoding = zc.ZapEncodeMethod
	} else if zc.ZapEncodeMethod == "json" {
		zapConfig.Encoding = zc.ZapEncodeMethod
	} else {
		zapConfig.Encoding = "console"
	}
	return &zapConfig
}

func Info(msg string, fields ...zap.Field) {
	logger.Info(msg, fields...)
}

func Infof(msg string, args ...interface{}) {
	sugarLogger.Infof(msg, args...)
}

func Infoff(ctx context.Context, msg string, args ...interface{}) {
	tracing := fmt.Sprintf(" trace_id=%s", trace.SpanFromContext(ctx).SpanContext().TraceID())
	msg = msg + tracing
	sugarLogger.Infof(msg, args...)
}

func Debug(msg string, fields ...zap.Field) {
	logger.Debug(msg, fields...)
}

func Debugf(msg string, args ...interface{}) {
	sugarLogger.Debugf(msg, args...)
}

func Warnf(msg string, args ...interface{}) {
	sugarLogger.Warnf(msg, args...)
}

func Error(msg string, fields ...zap.Field) {
	logger.Error(msg, fields...)
}

func Errorf(msg string, args ...interface{}) {
	sugarLogger.Errorf(msg, args...)
}

func Fatal(msg string, fields ...zap.Field) {
	logger.Fatal(msg, fields...)
}

func Fatalf(msg string, args ...interface{}) {
	sugarLogger.Fatalf(msg, args...)
}

func Panic(msg string, fields ...zap.Field) {
	logger.Panic(msg, fields...)
}

func Panicf(msg string, args ...interface{}) {
	sugarLogger.Panicf(msg, args...)
}

func Print(msg string, fields ...zap.Field) {
	logger.Info(msg, fields...)
}

func Printf(msg string, args ...interface{}) {
	sugarLogger.Infof(msg, args...)
}

func Println(msg string, fields ...zap.Field) {
	logger.Info(msg, fields...)
}

func Printlnf(msg string, args ...interface{}) {
	sugarLogger.Infof(msg, args...)
}
