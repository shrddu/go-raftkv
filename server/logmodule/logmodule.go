package logmodule

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"github.com/rosedblabs/rosedb/v2"
	"go-raftkv/common/log"
	"go-raftkv/common/rpc"
	"sync"
)

var LAST_INDEX_KEY = "LAST_INDEX_KEY"
var Log = log.GetLog()

type LogModule struct {
	DBDir   string
	LogsDir string
	DB      *rosedb.DB
	lock    *sync.Mutex
}

func (module *LogModule) Init() {
	module.lock = &sync.Mutex{}
}
func (module *LogModule) Destroy() {
	err := module.DB.Close()
	if err != nil {
		log.GetLog().Error(err)
		return
	}
	log.GetLog().Infof("logmodule.DB destroy success")
}

// Write
//
//	@Description: 写入entry到日志模块里
//	@receiver module
//	@param entry
func (module *LogModule) Write(entry *rpc.LogEntry) {
	module.lock.Lock()
	/*write方法中的data参数如果是结构体，那么结构体里不能有string类型。因为string是不定长类型，不支持string
	  所以选择使用gob包来编解码结构体，参考：https://blog.csdn.net/weixin_41896770/article/details/128152614
	*/
	//err := binary.Write(keyBuf, binary.LittleEndian, entry.Index)
	//if err != nil {
	//	Log.Error(err)
	//}
	//entryBuf := &bytes.Buffer{}
	//err = binary.Write(entryBuf, binary.LittleEndian, entry)
	//if err != nil {
	//	Log.Error(err)
	//}
	//err = module.DB.Put(keyBuf.Bytes(), entryBuf.Bytes())
	//if err != nil {
	//	Log.Error(err)
	//}
	/*使用gob编码结构体为字节数组*/
	keyBuf := &bytes.Buffer{}
	entryBuf := &bytes.Buffer{}
	if err := gob.NewEncoder(keyBuf).Encode(entry.Index); err != nil {
		Log.Error(err)
		return
	}
	if err := gob.NewEncoder(entryBuf).Encode(entry); err != nil {
		Log.Error(err)
		return
	}
	if err := module.DB.Put(keyBuf.Bytes(), entryBuf.Bytes()); err != nil {
		Log.Error(err)
		return
	}
	Log.Infof("logmodule write success : %+v", entry)
	module.UpdateLastIndex(entry.Index)
	module.lock.Unlock()
}
func (module *LogModule) Get(index int64) *rpc.LogEntry {
	indexBuf := new(bytes.Buffer)
	if err := gob.NewEncoder(indexBuf).Encode(index); err != nil {
		Log.Error(err)
		return nil
	}
	value, err := module.DB.Get(indexBuf.Bytes())
	if err != nil {
		Log.Error(err)
		return nil
	}
	valueBuf := bytes.NewBuffer(value)
	entry := &rpc.LogEntry{}
	if err := gob.NewDecoder(valueBuf).Decode(entry); err != nil {
		Log.Error(err)
		return nil
	}
	Log.Infof("read a entry %+v", entry)
	//u := binary.LittleEndian.Uint64(value)
	return entry
}

func (module *LogModule) DeleteOnStartIndex(startIndex int64) error {
	endIndex := module.GetLastIndex()
	for index := startIndex; index <= endIndex; index++ {
		indexBuf := &bytes.Buffer{}
		if err := gob.NewEncoder(indexBuf).Encode(index); err != nil {
			return err
		}
		err := module.DB.Delete(indexBuf.Bytes())
		if err != nil {
			return err
		}
	}
	Log.Infof("delete entries from %d Index has been success", startIndex)
	return nil
}
func (module *LogModule) GetLastIndex() int64 {
	value, err := module.DB.Get([]byte(LAST_INDEX_KEY))
	if err != nil {
		Log.Error(err)
		return 0
	}
	return int64(binary.BigEndian.Uint64(value))
}
func (module *LogModule) GetLastEntry() *rpc.LogEntry {
	value, err := module.DB.Get([]byte(LAST_INDEX_KEY))
	if err != nil {
		Log.Error(err)
		return nil
	}
	index := int64(binary.BigEndian.Uint64(value))
	entry := module.Get(index)
	return entry
}
func (module *LogModule) UpdateLastIndex(index int64) {
	module.lock.Lock()
	indexBuf := &bytes.Buffer{}
	if err := gob.NewEncoder(indexBuf).Encode(index); err != nil {
		Log.Error(err)
		return
	}
	if err := module.DB.Put([]byte(LAST_INDEX_KEY), indexBuf.Bytes()); err != nil {
		Log.Error(err)
		return
	}
	module.lock.Unlock()
	Log.Infoln("update LastIndex success")
}
