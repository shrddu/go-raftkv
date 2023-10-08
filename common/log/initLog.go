package log

import (
	"fmt"
	"go.uber.org/zap"
	"sync"
)

var once sync.Once
var instance *zap.SugaredLogger

func GetLog() *zap.SugaredLogger {
	once.Do(func() {
		logger, _ := zap.NewDevelopment()
		defer func() {
			err := logger.Sync()
			if err != nil {
				fmt.Println(err)
			}
		}() // flushes buffer, if any
		instance = logger.Sugar()
		instance.Infof("success to init zapLog")
	},
	)
	return instance
}
