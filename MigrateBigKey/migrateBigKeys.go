package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"time"
)

func usage(cmd string) {
	fmt.Printf("%s BigKeyFile SoureRedis DestRedis SlotId\n", cmd)
	os.Exit(2)
}

func InitRedisPool(redisAddr string) *redis.Pool {
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

var SrcPool *redis.Pool
var DstPool *redis.Pool

func InitAllPool() int {
	SrcPool = InitRedisPool(SrcRedis)
	if SrcPool == nil {
		GLogger.Printf("Init src redis pool [%s] failed.\n", SrcRedis)
		return 2
	}
	DstPool = InitRedisPool(DstRedis)
	if DstPool == nil {
		SrcPool.Close()
		GLogger.Printf("Init dst redis pool [%s] failed.\n", DstRedis)
		return 2
	}
	return 0
}

func ReleasePool() {
	if SrcPool != nil {
		SrcPool.Close()
	}
	if DstPool != nil {
		DstPool.Close()
	}
}

var GredisCmd string = "/letv/codis/bin/redis-cli -p 6381 "

func execResult(cmdstr string) (string, error) {
	cmd := exec.Command("/bin/sh", "-c", cmdstr)
	if cmd == nil {
		GLogger.Printf("exec command [%s] failed.\n", cmdstr)
		return "", errors.New("command failed")
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		GLogger.Printf("StdoutPipe failed,err:%s.\n", err.Error())
		return "", err
	}
	e := cmd.Start()
	if e != nil {
		GLogger.Printf("cmd Start failed,err:%s.\n", e.Error())
		return "", e
	}
	bytes, eio := ioutil.ReadAll(stdout)
	if eio != nil {
		GLogger.Printf("ReadAll failed,err:%s\n", eio.Error())
		return "", eio
	}
	e = cmd.Wait()
	if e != nil {
		GLogger.Printf("Wait failed,err:%s\n", e.Error())
		return "", e
	}
	return string(bytes), nil
}

func KeyFieldMig(src redis.Conn, dst redis.Conn, key string, keytype string, field string) error {
	//fmt.Printf("Now migr key [%s]\n", key)
	var srcReply interface{}
	var dstReply interface{}
	var err error
	switch keytype {
	case "hash":
		srcReply, err = src.Do("hget", key, field)
		if err != nil {
			GLogger.Printf("hget %s %s failed,err:%s", key, field, err.Error())
			return err
		}
		if srcReply != nil {
			dstReply, err = dst.Do("hset", key, field, srcReply)
			if err != nil {
				GLogger.Printf("hset %s %s [%v] failed,err:%s", key, field, srcReply, err.Error())
				return err
			}
			GLogger.Printf("hset %s %s, reply:%v", key, field, dstReply)
		}
	case "list":

	case "set":

	case "zset":

	default:
		return nil
	}
	return nil
}

func Migrate(src redis.Conn, dst redis.Conn, slotIndex int, key string, keytype string) {
	var cmdstr string
	var tmpfilename string

	// get all field of one key
	switch keytype {
	case "hash":
		tmpfilename = key + "." + keytype + ".tmpfield"
		cmdstr = GredisCmd + "hkeys " + key + " > " + tmpfilename
	case "list":
		cmdstr = ""
	case "set":
		cmdstr = ""
	case "zset":
		cmdstr = ""
	}
	GLogger.Printf("cmd is:[%s]\n", cmdstr)
	_, err := execResult(cmdstr)
	if err != nil {
		GLogger.Printf("exec failed,err:%s\n", err.Error())
		return
	}
	// read all field
	f, e := os.Open(tmpfilename)
	if e != nil {
		GLogger.Printf("open file [%s] failed,err:%s\n", e.Error())
		return
	}
	defer f.Close()
	fmt.Printf("now begin migrate key [%s]\n", key)
	var number int = 0
	rd := bufio.NewReader(f)
	for {
		line, eline := rd.ReadString('\n')
		if eline != nil || io.EOF == err {
			break
		}
		line = strings.Trim(line, "\n")
		number++
		if number%5000 == 0 {
			fmt.Printf("[%s] migrate key [%s] field number [%d]\n", time.Now().String(), key, number)
		}
		KeyFieldMig(src, dst, key, keytype, line)
	}
	/*var delReply interface{}
	delReply, err = src.Do("del", key)
	if err != nil {
		GLogger.Printf("del %s failed,err:%s", key, err.Error())
	}
	if delReply != nil {
		GLogger.Printf("del %s success.")
	}*/
	fmt.Printf("you should migrate slot[%d] from [%s] to [%s]\n", slotIndex, SrcRedis, DstRedis)
	return
}

func MigrateSlot(filename string, slotIndex int) {
	var slotindex int
	var key string
	var keytype string
	var tmp1, tmp2, tmp3 int

	// open big key file
	fp, err := os.Open(filename)
	if err != nil {
		GLogger.Printf("Open file [%s] failed.\n", filename)
		return
	}
	defer fp.Close()
	// get pool connect
	SrcConn := SrcPool.Get()
	DstConn := DstPool.Get()
	if SrcConn == nil || DstConn == nil {
		GLogger.Printf("Get Conn failed\n")
		return
	}
	defer SrcConn.Close()
	defer DstConn.Close()

	// read every key
	rd := bufio.NewReader(fp)
	for {
		line, err := rd.ReadString('\n')
		if err != nil || io.EOF == err {
			break
		}
		line = strings.Trim(line, "\n")
		n, err := fmt.Sscanf(line, "%d\t%s\t%s\t%d\t%d\t%d\n", &slotindex, &key, &keytype, &tmp1, &tmp2, &tmp3)
		if err != nil || n != 6 {
			GLogger.Printf("Sscanf failed,n is:%d,err:%s.\n", n, err.Error())
			continue
		}
		// slot one big key
		if slotIndex == slotindex {
			Migrate(SrcConn, DstConn, slotindex, key, keytype)
		}
	}
}

var KeysFile string
var SrcRedis string
var DstRedis string
var SlotId int64
var LogFileName = "MigrateBigKey.log"
var GLogger *log.Logger

func InitLog() {
	file, err := os.OpenFile(LogFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, os.ModePerm)
	if err != nil {
		fmt.Printf("Create log file [%s] failed,err:%s\n", LogFileName, err.Error())
		os.Exit(2)
	}
	GLogger = log.New(file, "", log.LstdFlags)
	if GLogger == nil {
		fmt.Printf("New log failed.\n")
		os.Exit(2)
	}
}

func main() {
	var err error
	c := make(chan os.Signal, 1)
	if len(os.Args) != 5 {
		usage(os.Args[0])
	}
	// parse args
	LogFileName = os.Args[0] + ".log"
	KeysFile = os.Args[1]
	SrcRedis = os.Args[2]
	DstRedis = os.Args[3]
	SlotId, err = strconv.ParseInt(os.Args[4], 10, 32)
	if err != nil {
		fmt.Printf("invalid slotId:%s\n", os.Args[4])
		usage(os.Args[0])
	}
	// register interrupt signal
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		ReleasePool()
		fmt.Printf("recv SIGINTERRUPT,now exit.\n")
		os.Exit(2)
	}()
	// init log
	InitLog()
	// init redis conn pool
	if InitAllPool() != 0 {
		fmt.Printf("InitAllPool failed.\n")
		return
	}
	//begin migrate big keys
	MigrateSlot(KeysFile, int(SlotId))
	ReleasePool()
}
