package app

import "github.com/davidtrse/graceful/kafkas"

var Instance *Context

type Context struct {
	KafkaManager           kafkas.IKafkaManager
	GracefulShutDownManage GracefulShutDownManage
}
