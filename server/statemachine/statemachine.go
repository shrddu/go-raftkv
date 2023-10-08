package statemachine

import (
	"encoding/json"
	"github.com/rosedblabs/rosedb/v2"
	"go-raftkv/common/log"
	"go-raftkv/common/rpc"
	"go.uber.org/zap"
	"sync"
)

var BasePath = "D:/GoStudy/GOPATH/src/go-raftkv/tmp/rosedb_basic/"
var Log *zap.SugaredLogger

type StateMachine struct {
	RSDB *rosedb.DB
}

var instance *StateMachine
var once sync.Once

func GetInstance(selfPort string) *StateMachine {
	once.Do(func() {
		instance = &StateMachine{}
		instance.Init(selfPort)
	})
	return instance
}

func (machine *StateMachine) Init(selfPort string) {
	// 指定选项
	options := rosedb.DefaultOptions
	options.DirPath = BasePath + selfPort + "/statemachine/"
	Log = log.GetLog()
	// 打开数据库
	db, err := rosedb.Open(options)
	if err != nil {
		Log.Error(err)
	}

	machine.RSDB = db
}

func (machine *StateMachine) Destroy() {
	err := machine.RSDB.Close()
	if err != nil {
		return
	}
}

func (machine *StateMachine) Set(entry *rpc.LogEntry) bool {
	entryBytes, _ := json.Marshal(entry)
	err := machine.RSDB.Put([]byte(entry.K), entryBytes)
	if err != nil {
		panic(err)
	}
	Log.Infof("success to set a k and v : %s : %+v", entry.K, entry)
	return true
}

func (machine *StateMachine) Get(key string) *rpc.LogEntry {
	val, err := machine.RSDB.Get([]byte(key))
	if err != nil {
		Log.Error(err)
		return nil
	}
	result := rpc.LogEntry{}
	err = json.Unmarshal(val, &result)
	if err != nil {
		Log.Errorf("fail to Get : %v", err)
		return nil
	}
	Log.Infof("success to get value : %+v", result)
	return &result
}

func (machine *StateMachine) Del(key string) bool {
	err := machine.RSDB.Delete([]byte(key))
	if err != nil {
		panic(err)
	}
	Log.Infof("success to Del a entry,key = %s:", key)
	return true
}
