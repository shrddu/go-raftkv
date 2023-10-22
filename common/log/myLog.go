package log

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"sync"
)

var once sync.Once
var instance *zap.SugaredLogger

func GetLog() *zap.SugaredLogger {
	once.Do(func() {
		// 第一种方式
		//logger, _ := zap.NewDevelopment()

		//第二种方式
		//encoderCfg := zap.NewProductionEncoderConfig()
		//encoderCfg.TimeKey = "timestamp"
		//encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
		//config := zap.Config{
		//	Level:             zap.NewAtomicLevelAt(zap.InfoLevel),
		//	Development:       false,
		//	DisableCaller:     false,
		//	DisableStacktrace: false,
		//	Sampling:          nil,
		//	Encoding:          "json",
		//	EncoderConfig:     encoderCfg,
		//	OutputPaths: []string{
		//		"stderr",
		//	},
		//	ErrorOutputPaths: []string{
		//		"stderr",
		//	},
		//	InitialFields: map[string]interface{}{
		//		"pid": os.Getpid(),
		//	},
		//}
		//logger := zap.Must(config.Build())

		// 第三种方式 基准测试或者生产环境下用这种
		file, _ := os.Create("./test.log")
		writeSyncer := zapcore.AddSync(file)
		encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
		core := zapcore.NewCore(encoder, writeSyncer, zapcore.WarnLevel)
		logger := zap.New(core)

		defer func() {
			err := logger.Sync()
			if err != nil {
				fmt.Println(err)
			}
		}() // flushes buffer

		instance = logger.Sugar()
		instance.Infof("success to init zapLog")
	},
	)
	return instance
}
