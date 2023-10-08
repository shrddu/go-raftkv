package main

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"time"
)

var Log zap.SugaredLogger

func main() {
	initModule()
	go func(t *time.Ticker) {
		for range t.C {
			fmt.Println(time.Now().String() + "6666")
		}
	}(time.NewTicker(time.Second))
	time.Sleep(1 * time.Minute)

}

func initModule() {
	initLogger()
}

func initLogger() {
	logger, _ := zap.NewDevelopment(zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	Log = *(logger.Sugar())
	defer Log.Sync()
}
