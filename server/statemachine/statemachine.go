package statemachine

import (
	"bytes"
	"encoding/gob"
	"github.com/rosedblabs/rosedb/v2"
	"go-raftkv/common/log"
	"go-raftkv/common/rpc"
	"go.uber.org/zap"
	"sync"
)

// TODO 设置为相对地址
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
	Log.Infof("node %s get a stateMachine %+v", "localhost:"+selfPort, instance)
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

// Set
//
//	@Description: 保存到状态机 entry.K(string) : entry(LogEntry) 的格式
//	@receiver machine
//	@param entry
//	@return bool
func (machine *StateMachine) Set(entry *rpc.LogEntry) bool {
	// SetInt64(int64) 参数如果是0的话，会自动转换为nil，所以设置失效，不能使用big.Int{}.setInt64()这种方法
	if entry.K == "" {
		Log.Infof("stateMachine doesn't apply empty 'K' entry , method is return")
		return true
	}
	keyBuf := &bytes.Buffer{}
	entryBuf := &bytes.Buffer{}
	if err := gob.NewEncoder(keyBuf).Encode(entry.K); err != nil {
		Log.Debug(err)
		return false
	}
	if err := gob.NewEncoder(entryBuf).Encode(entry); err != nil {
		Log.Debug(err)
		return false
	}
	err := machine.RSDB.Put(keyBuf.Bytes(), entryBuf.Bytes())
	if err != nil {
		panic(err)
	}
	Log.Infof("stateMachine : success to apply a entry : %+v", entry)
	return true
}

// Get
//
//	@Description: 传入entry.K 返回 LogEntry
//	@receiver machine
//	@param key
//	@return *rpc.LogEntry
func (machine *StateMachine) Get(key string) *rpc.LogEntry {
	keyBuf := &bytes.Buffer{}
	if err := gob.NewEncoder(keyBuf).Encode(key); err != nil {
		Log.Warn(err)
		return nil
	}
	val, err := machine.RSDB.Get(keyBuf.Bytes())
	if err != nil {
		Log.Error(err)
		return nil
	}
	valueBuf := bytes.NewBuffer(val)
	entry := &rpc.LogEntry{}
	if err := gob.NewDecoder(valueBuf).Decode(entry); err != nil {
		Log.Warn(err)
		return nil
	}
	Log.Infof("success to get value : %+v", entry)
	return entry
}

func (machine *StateMachine) Del(key string) bool {
	err := machine.RSDB.Delete([]byte(key))
	if err != nil {
		panic(err)
	}
	Log.Infof("success to Del a entry,key = %s:", key)
	return true
}
