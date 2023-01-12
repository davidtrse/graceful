package log

import "go.uber.org/zap"

type ILog interface {
	Info(string, ...zap.Field)
	Infof(string, ...interface{})
	Infoff(traceId string, msg string, args ...interface{})
	Debugf(string, ...interface{})
	Error(string, ...zap.Field)
	Errorf(string, ...interface{})
	Fatal(string, ...zap.Field)
	Fatalf(...interface{})
	Panic(string, ...zap.Field)
	Panicf(string, ...interface{})
	Print(string, ...zap.Field)
	Printf(string, ...interface{})
	Println(string, ...zap.Field)
	Printlnf(string, ...interface{})
}
