package main

import (
	md "../codis/pkg/models"
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/wandoulabs/zkhelper"
	"hash/crc32"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var MaxSlotNum uint32 = 1024

func hashSlot(key []byte) int {
	const (
		TagBeg = '{'
		TagEnd = '}'
	)
	if beg := bytes.IndexByte(key, TagBeg); beg >= 0 {
		if end := bytes.IndexByte(key[beg+1:], TagEnd); end >= 0 {
			key = key[beg+1 : beg+1+end]
		}
	}
	return int(crc32.ChecksumIEEE(key) % MaxSlotNum)
}

var Slog *log.Logger
var LogFile *os.File

type SlotsInfo struct {
	zkConn      zkhelper.Conn
	zkAddr      string
	productName string
	sinfo       map[int]int
	failflag    [1024]bool
	slogname    string
}

var AllSlots SlotsInfo = SlotsInfo{
	zkAddr:      "10.98.28.5:2181",
	productName: "pusher",
	sinfo:       make(map[int]int),
	slogname:    "checkSlotKeys.log",
}

func confSlots(logFileName string, zkAddr string, dbName string) {
	AllSlots.slogname = (logFileName + ".log")
	AllSlots.zkAddr = zkAddr
	AllSlots.productName = dbName
}

func initLog() error {
	var err error
	LogFile, err = os.Create(AllSlots.slogname)
	if err != nil {
		return errors.New("Create log file failed")
	}
	Slog = log.New(LogFile, "", log.LstdFlags)
	if Slog == nil {
		return errors.New("log New failed")
	}
	Slog.Println("initLog success")
	return nil
}

func getSlotInfo(zkConn zkhelper.Conn, productName string, slotid uint32) (*md.Slot, error) {
	s, err := md.GetSlot(zkConn, productName, int(slotid))
	if err != nil {
		fmt.Printf("get slot [%d] failed\n", slotid)
		return nil, errors.New("get slot failed.")
	}
	return s, nil
}

func getSlots() error {
	var err error
	AllSlots.zkConn, err = zkhelper.ConnectToZk(AllSlots.zkAddr, 30000)
	if err != nil {
		fmt.Printf("connect to zk [%s] failed\n", AllSlots.zkAddr)
		return err
	}
	var i uint32 = 0
	for i = 0; i < MaxSlotNum; i++ {
		s, serr := getSlotInfo(AllSlots.zkConn, AllSlots.productName, i)
		if serr != nil {
			return serr
		}
		AllSlots.sinfo[s.Id] = s.GroupId
		Slog.Printf("slot index [%d],group id [%d]", s.Id, s.GroupId)
	}
	AllSlots.zkConn.Close()
	return nil
}

func printSlotInfo() {
	var i uint32 = 0
	for i = 0; i < MaxSlotNum; i++ {
		fmt.Printf("slot id [%d],group id [%d]\n", i, AllSlots.sinfo[int(i)])
	}
}

var RedisAddr string
var GroupId int64
var RedisPool *redis.Pool

func RedisPoolInit() *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", RedisAddr)
			if err != nil {
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

func checkInvalidData(c redis.Conn, key string, keytype string, slotId int) {
	if AllSlots.sinfo[slotId] != int(GroupId) {
		Slog.Printf("key [%s], type [%s], slot [%d] is not in group[%d],may be invalid data.",
			key, keytype, slotId, GroupId)
	} else {
		return
	}
	_, err := c.Do("DEL", key)
	if err != nil {
		Slog.Printf("DEL %s failed,err:%s", key, err.Error())
	} else {
		Slog.Printf("DEL %s success", key)
	}
}

func execResult(cmdstr string) string {
	cmd := exec.Command("/bin/sh", "-c", cmdstr)
	if cmd == nil {
		Slog.Printf("exec [%s] failed", cmdstr)
		return ""
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		Slog.Printf("StdoutPipe failed")
		return ""
	}
	e := cmd.Start()
	if e != nil {
		Slog.Printf("start cmd failed,err:%s", e.Error())
		return ""
	}
	bytes, eio := ioutil.ReadAll(stdout)
	if eio != nil {
		Slog.Printf("readl all failed,err:%s", eio.Error())
		return ""
	}
	e = cmd.Wait()
	if e != nil {
		Slog.Printf("cmd wait failed,err:%s", e.Error())
		return ""
	}
	return string(bytes)
}

var GredisCmd string = "/letv/codis/bin/redis-cli -p 6381 "

func getPerFieldLen(c redis.Conn, key string, keytype string) int {
	var pervaluelen int = 0
	switch keytype {
	case "hash":
		cmdstr := GredisCmd + "hscan " + key + " 0 count 1"
		Slog.Printf("now will hscan key[%s],cmd[%s]", key, cmdstr)
		res := execResult(cmdstr)
		//Slog.Printf("res:%s", res)
		reslen := len(res)
		if reslen > 0 {
			pervaluelen = (reslen - len(key)) / 2
		}
	case "list":
		cmdstr := GredisCmd + "lindex " + key + " 0"
		Slog.Printf("now will lindex key[%s],cmd[%s]", key, cmdstr)
		res := execResult(cmdstr)
		reslen := len(res)
		if reslen > 0 {
			pervaluelen = reslen
		}
	case "set":
		cmdstr := GredisCmd + "sscan " + key + " 0 count 1"
		Slog.Printf("now will sscan key[%s],cmd[%s]", key, cmdstr)
		res := execResult(cmdstr)
		reslen := len(res)
		if reslen > 0 {
			pervaluelen = reslen
		}
	case "zset":
		cmdstr := GredisCmd + "zscan " + key + " 1 count 1"
		Slog.Printf("now will zscan key[%s],cmd[%s]", key, cmdstr)
		res := execResult(cmdstr)
		reslen := len(res)
		if reslen > 0 {
			pervaluelen = reslen / 4
		}

	default:
		return 0

	}

	return pervaluelen
}

func saveBigKeys(slotIndex int, key string, keytype string, fieldlen int, pervaluelen int) {
	addr := strings.Split(RedisAddr, ":")
	var filename string = "./" + addr[0] + ".bigkeys"

	fp, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, os.ModePerm)
	if err != nil {
		Slog.Printf("open file [%s] failed,err:%s\n", filename, err.Error())
		return
	}
	defer fp.Close()
	keyinfo := fmt.Sprintf("%d\t%s\t%s\t%d\t%d\t%d\n", slotIndex, key, keytype, fieldlen, pervaluelen, fieldlen*pervaluelen)
	fp.WriteString(keyinfo)
	return
}

func checkPerKey(c redis.Conn, key string) error {
	keytype, err := c.Do("TYPE", key)
	if err != nil {
		Slog.Printf("Do TYPE [%s] failed,err:%s", key, err.Error())
		return errors.New("get type of key failed")
	}
	var reply interface{}
	var keylen int64
	switch keytype {
	case "string":
		reply, err = c.Do("STRLEN", key)
	case "list":
		reply, err = c.Do("LLEN", key)
	case "set":
		reply, err = c.Do("SCARD", key)
	case "zset":
		reply, err = c.Do("ZCARD", key)
	case "hash":
		reply, err = c.Do("HLEN", key)
	case "none":
		Slog.Printf("key [%s] type [none], may be deleted.", key)
	default:
		Slog.Printf("key [%s] type [%s] is invalid", key, keytype)
		return errors.New("key type is invalid")
	}
	if keytype == "none" {
		return nil
	}

	var slotIndex int = hashSlot([]byte(key))
	checkInvalidData(c, key, keytype.(string), slotIndex)

	if err != nil {
		Slog.Printf("key [%s] get len failed,err:%s", key, err.Error())
		return errors.New("get key len failed")
	}
	if reply == nil {
		Slog.Printf("key [%s],type [%s], value is nil, slot [%d] can migrate.", key, keytype, hashSlot([]byte(key)))
		return nil
	}
	switch reply.(type) {
	case string:
		{
			keylen, err = strconv.ParseInt(reply.(string), 10, 64)
			if err != nil {
				Slog.Printf("key [%s] invalid keylen [%s]", key, reply)
				return errors.New("parse int failed")
			}
		}
	case int64:
		keylen = reply.(int64)
	default:
		keylen = reply.(int64)
	}

	if keylen > 4096 {
		Slog.Printf("key [%s],type [%s], len is [%d],too long,slot [%d] should not migrate", key, keytype, keylen, slotIndex)
		AllSlots.failflag[slotIndex] = true
		pervaluelen := getPerFieldLen(c, key, keytype.(string))
		Slog.Printf("get key[%s],type[%s],pervalue len[%d]", key, keytype, pervaluelen)
		if pervaluelen > 0 {
			saveBigKeys(slotIndex, key, keytype.(string), int(keylen), pervaluelen)
		}
		return nil
		//return errors.New("key's len is too long")
	}
	Slog.Printf("key [%s],type [%s], len is [%d],slot [%d] can migrate", key, keytype, keylen, slotIndex)
	return nil
}

func checkKeysFile(filename string) error {
	RedisPool = RedisPoolInit()
	if RedisPool == nil {
		fmt.Printf("init RedisPool failed")
		return errors.New("poll init failed")
	}
	Slog.Println("redis RedisPool init success")
	defer RedisPool.Close()
	f, err := os.Open(filename)
	if err != nil {
		fmt.Printf("open file [%s] failed\n", filename)
		return errors.New("open file failed")
	}
	defer f.Close()
	rd := bufio.NewReader(f)
	var checkNum int32 = 0
	Slog.Println("now begin check kyes...")

	c := RedisPool.Get()
	if c == nil {
		fmt.Printf("get poll failed")
		return errors.New("get client from RedisPool failed.")
	}
	defer c.Close()
	for {
		line, err := rd.ReadString('\n')
		if err != nil || io.EOF == err {
			break
		}
		line = strings.Trim(line, "\n")

		checkNum += 1
		if (checkNum % 5000) == 0 {
			fmt.Printf("now [%v] check keys num [%v]\n", time.Now(), checkNum)
		}
		//fmt.Printf("now check key[%s]\n", line)
		err = checkPerKey(c, line)
		if err != nil {
			Slog.Printf("check key [%s], slot [%d] failed,err:%s", line, hashSlot([]byte(line)), err.Error())
			continue
		}
	}
	return nil
}

func printSlotCheckResult() {
	//fmt.Printf("map int bool len:%d\n", len(AllSlots.failflag))
	var mgrtSlots [1024]int
	var i int = 0
	var j int = 0

	for i = 0; i < 1024; i++ {
		mgrtSlots[i] = -1
	}
	for i = 0; i < len(AllSlots.failflag); i++ {
		if AllSlots.sinfo[i] == int(GroupId) && AllSlots.failflag[i] != true {
			Slog.Printf("slot [%d] can be migrated", i)
			fmt.Printf("slot [%d] can be migrated\n", i)
			mgrtSlots[j] = i
			j++
		}
	}
	fmt.Printf("can migrate slot index: \nbeg  end\n")
	var newFlag bool = false
	for i = 0; i < j; i++ {
		if i == 0 || i == (j-1) || newFlag == true {
			if mgrtSlots[i] != -1 {
				if newFlag == true {
					fmt.Printf("\n")
				}
				fmt.Printf("%04d ", mgrtSlots[i])
			}
		} else {
			if mgrtSlots[i+1]-mgrtSlots[i] > 1 {
				newFlag = true
				fmt.Printf("%04d ", mgrtSlots[i])
			}
		}
		if mgrtSlots[i+1]-mgrtSlots[i] == 1 {
			newFlag = false
		}
	}
	fmt.Printf("\n\n")
	for i = 0; i < len(AllSlots.failflag); i++ {
		if AllSlots.sinfo[i] == int(GroupId) && AllSlots.failflag[i] == true {
			Slog.Printf("slot [%d] can not be migrated", i)
			//fmt.Printf("slot [%d] can not be migrated\n", i)
		}
	}
}

func usage(pragramName string) {
	fmt.Printf("%s keyfile ZkAddr DbName RedisAddr GroupId\n", pragramName)
	fmt.Printf(" keyfile is a file contains all keys of redis server.you can get it by:\n")
	fmt.Printf(" /letv/codis/bin/redis-cli -p 6381 keys '*' > 10.98.28.25.keys\n")
	fmt.Printf("  eg: ./checkSlot 10.98.28.25.keys 10.98.28.5:2181 pusher 10.98.28.26:6381 2\n")
}

func main() {
	var err error
	// check parameters
	if len(os.Args) != 6 {
		usage(os.Args[0])
		os.Exit(0)
	}
	keyFileName := os.Args[1]
	zkAddr := os.Args[2]
	dbName := os.Args[3]
	RedisAddr = os.Args[4]
	gid := os.Args[5]
	GroupId, err = strconv.ParseInt(gid, 10, 32)
	if err != nil {
		usage(os.Args[0])
		fmt.Printf("GroupId must be a int number\n")
		os.Exit(0)
	}
	fmt.Printf("KeyFileName[%s],ZkAddr[%s],DbName:[%s],RedisAddr[%s],GroupId [%d]\n",
		keyFileName, zkAddr, dbName, RedisAddr, GroupId)
	// configure slots information
	confSlots(os.Args[0], zkAddr, dbName)
	// initialize logfile
	err = initLog()
	if err != nil {
		fmt.Printf("init log failed,err:%s", err.Error())
		return
	}
	// get slot group information from zookeeper
	err = getSlots()
	if err != nil {
		fmt.Printf("get slot info failed,err:%s\n", err.Error())
		return
	}
	//printSlotInfo()
	// check slot can be migrated by key file
	err = checkKeysFile(keyFileName)
	if err != nil {
		return
	}
	printSlotCheckResult()

}
