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
	"log"
	"os"
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
var logfile *os.File

type SlotsInfo struct {
	zkConn      zkhelper.Conn
	zkAddr      string
	productName string
	sinfo       map[int]int
	failflag    [1024]bool
	slogname    string
}

var allSlots SlotsInfo = SlotsInfo{
	zkAddr:      "10.98.28.5:2181",
	productName: "pusher",
	sinfo:       make(map[int]int),
	slogname:    "checkSlotKeys.log",
}

func initLog() error {
	var err error
	logfile, err = os.Create(allSlots.slogname)
	if err != nil {
		return errors.New("Create log file failed")
	}
	Slog = log.New(logfile, "", log.LstdFlags)
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
	allSlots.zkConn, err = zkhelper.ConnectToZk(allSlots.zkAddr, 30000)
	if err != nil {
		fmt.Printf("connect to zk [%s] failed\n", allSlots.zkAddr)
		return err
	}
	var i uint32 = 0
	for i = 0; i < MaxSlotNum; i++ {
		s, serr := getSlotInfo(allSlots.zkConn, allSlots.productName, i)
		if serr != nil {
			return serr
		}
		allSlots.sinfo[s.Id] = s.GroupId
		Slog.Printf("slot index [%d],group id [%d]", s.Id, s.GroupId)
	}
	allSlots.zkConn.Close()
	return nil
}

func printSlotInfo() {
	var i uint32 = 0
	for i = 0; i < MaxSlotNum; i++ {
		fmt.Printf("slot id [%d],group id [%d]\n", i, allSlots.sinfo[int(i)])
	}
}

var redisAddr string
var groupId int64
var pool *redis.Pool

func poolInit() *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", redisAddr)
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

func checkInvalidData(key string, keytype string, slotId int) {
	if allSlots.sinfo[slotId] != int(groupId) {
		Slog.Printf("key [%s], type [%s], slot [%d] is not in group[%d],may be invalid data.",
			key, keytype, slotId, groupId)
	}
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
	checkInvalidData(key, keytype.(string), slotIndex)

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
		allSlots.failflag[slotIndex] = true
		return nil
		//return errors.New("key's len is too long")
	}
	Slog.Printf("key [%s],type [%s], len is [%d],slot [%d] can migrate", key, keytype, keylen, slotIndex)
	return nil
}

func checkKeysFile(filename string) error {
	pool = poolInit()
	if pool == nil {
		fmt.Printf("init pool failed")
		return errors.New("poll init failed")
	}
	Slog.Println("redis pool init success")
	defer pool.Close()
	f, err := os.Open(filename)
	if err != nil {
		fmt.Printf("open file [%s] failed\n", filename)
		return errors.New("open file failed")
	}
	defer f.Close()
	rd := bufio.NewReader(f)
	var checkNum int32 = 0
	Slog.Println("now begin check kyes...")

	c := pool.Get()
	if c == nil {
		fmt.Printf("get poll failed")
		return errors.New("get client from pool failed.")
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
	//fmt.Printf("map int bool len:%d\n", len(allSlots.failflag))
	for i := 0; i < len(allSlots.failflag); i++ {
		if allSlots.sinfo[i] == int(groupId) && allSlots.failflag[i] != true {
			Slog.Printf("slot [%d] can be migrated", i)
			fmt.Printf("slot [%d] can be migrated\n", i)
		}
	}
	for i := 0; i < len(allSlots.failflag); i++ {
		if allSlots.sinfo[i] == int(groupId) && allSlots.failflag[i] == true {
			Slog.Printf("slot [%d] can not be migrated", i)
			fmt.Printf("slot [%d] can not be migrated\n", i)
		}
	}
}

func usage(pragramName string) {
	fmt.Printf("%s keyfile redisAddr groupId\n", pragramName)
}

func main() {
	if len(os.Args) != 4 {
		usage(os.Args[0])
		os.Exit(0)
	}

	err := initLog()
	if err != nil {
		fmt.Printf("init log failed,err:%s", err.Error())
		return
	}

	e := getSlots()
	if e != nil {
		fmt.Printf("get slot info failed,err:%s\n", err.Error())
		return
	}
	//printSlotInfo()

	filename := os.Args[1]
	redisAddr = os.Args[2]
	gid := os.Args[3]
	groupId, err = strconv.ParseInt(gid, 10, 32)
	if err != nil {
		usage(os.Args[0])
		fmt.Printf("groupId must be a int number\n")
		os.Exit(0)
	}
	fmt.Printf("filename [%s],redisAddr[%s],groupId [%d]\n", filename, redisAddr, groupId)
	e = checkKeysFile(filename)
	if e != nil {
		return
	}
	printSlotCheckResult()

}
