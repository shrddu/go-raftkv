package log

import (
	"go.uber.org/zap"
	"sync"
)

var once sync.Once
var instance *zap.SugaredLogger

func GetLog() *zap.SugaredLogger {
	once.Do(func() {
		logger, _ := zap.NewDevelopment()
		defer logger.Sync() // flushes buffer, if any
		instance = logger.Sugar()
		instance.Infof("success to init zapLog")
	},
	)
	return instance
}
