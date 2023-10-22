package logmodule

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"github.com/rosedblabs/rosedb/v2"
	"go-raftkv/common/log"
	"go-raftkv/common/rpc"
	"strconv"
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
		Log.Error(err)
		return
	}
	Log.Infof("logmodule.DB destroy success")
}

// Set
//
//	@Description: 写入entry到日志模块里，key:entry.Index(string) value:entry(LogEntry)
//	@receiver module
//	@param entry
func (module *LogModule) Set(entry *rpc.LogEntry) {
	if !module.lock.TryLock() {
		Log.Infof("logmodule get MutexLock failed ,set operation is failed")
		return
	}
	defer module.lock.Unlock()
	/*write方法中的data参数如果是结构体，那么结构体里不能有string类型。因为string是不定长类型，不支持string
	  所以选择使用gob包来编解码结构体，参考：https://blog.csdn.net/weixin_41896770/article/details/128152614
	*/
	/*使用gob编码结构体为字节数组*/
	keyBuf := &bytes.Buffer{}
	entryBuf := &bytes.Buffer{}
	indexString := strconv.FormatInt(entry.Index, 10)

	if err := gob.NewEncoder(keyBuf).Encode(indexString); err != nil {
		Log.Warn(err)
		return
	}
	if err := gob.NewEncoder(entryBuf).Encode(entry); err != nil {
		Log.Warn(err)
		return
	}
	if err := module.DB.Put(keyBuf.Bytes(), entryBuf.Bytes()); err != nil {
		Log.Warn(err)
		return
	}
	Log.Infof("logmodule write success : %+v", entry)
	if entry.Index != 0 {
		module.UpdateLastIndex(entry.Index)
	} else if module.GetLastIndex() == 0 {
		module.UpdateLastIndex(0)
	}
}

// Get
//
//	@Description: 读取logModule中的数据，
//	@receiver module
//	@param index 自动转换为string类型查询
//	@return *rpc.LogEntry
func (module *LogModule) Get(index int64) *rpc.LogEntry {
	indexBuf := new(bytes.Buffer)
	indexString := strconv.FormatInt(index, 10)
	if err := gob.NewEncoder(indexBuf).Encode(indexString); err != nil {
		Log.Debug(err)
		return nil
	}
	value, err := module.DB.Get(indexBuf.Bytes())
	if err != nil || value == nil {
		Log.Debug(err)
		return nil
	}
	valueBuf := bytes.NewBuffer(value)
	entry := &rpc.LogEntry{}
	if err := gob.NewDecoder(valueBuf).Decode(entry); err != nil {
		Log.Debug(err)
		return nil
	}
	Log.Infof("logModule read a entry %+v", entry)
	//u := binary.LittleEndian.Uint64(value)
	return entry
}
func (module *LogModule) Delete(index int64) error {
	indexBuf := &bytes.Buffer{}
	indexString := strconv.FormatInt(index, 10)
	if err := gob.NewEncoder(indexBuf).Encode(indexString); err != nil {
		return err
	}
	err := module.DB.Delete(indexBuf.Bytes())
	if err != nil {
		Log.Errorf("delete entry which index is : %d failed", index)
		return err
	}
	Log.Infof("delete entry which index is : %d success", index)
	return nil
}

// DeleteOnStartIndex
//
//	@Description: 删除startIndex以后的log
//	@receiver module
//	@param startIndex
//	@return error
func (module *LogModule) DeleteOnStartIndex(startIndex int64) error {
	endIndex := module.GetLastIndex()
	for index := startIndex; index <= endIndex; index++ {
		indexBuf := &bytes.Buffer{}
		indexString := strconv.FormatInt(index, 10)
		if err := gob.NewEncoder(indexBuf).Encode(indexString); err != nil {
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

// GetLastIndex
//
//	@Description: logModule中最后index "LAST_INDEX_KEY"(string):index(int64)
//	@receiver module
//	@return int64
func (module *LogModule) GetLastIndex() int64 {
	value, err := module.DB.Get([]byte(LAST_INDEX_KEY))
	if err != nil {
		Log.Debug(err)
		return 0
	}
	lastIndex := bytesToInt64(value)
	Log.Infof("get lastIndex : %d", lastIndex)
	return lastIndex
}

func (module *LogModule) GetLastEntry() *rpc.LogEntry {
	value, err := module.DB.Get([]byte(LAST_INDEX_KEY))
	if err != nil {
		Log.Debug(err)
		return nil
	}
	index := bytesToInt64(value)
	entry := module.Get(index)
	return entry
}

// UpdateLastIndex
//
//	@Description: 更新lastIndex日志，"LAST_INDEX_KEY"(string):index(int64)
//	@receiver module
//	@param index
func (module *LogModule) UpdateLastIndex(index int64) {
	// 上层已经上锁了
	bts := int64ToBytes(index)
	// int64转string 存储的类型是:  K:string V:string
	if err := module.DB.Put([]byte(LAST_INDEX_KEY), bts); err != nil {
		Log.Debug(err)
		return
	}
	Log.Infof("'LastIndex' update to %d success", index)
}
func int64ToBytes(i int64) []byte {
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(i))
	return buf
}
func bytesToInt64(buf []byte) int64 {
	return int64(binary.BigEndian.Uint64(buf))
}
