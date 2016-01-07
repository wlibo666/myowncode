package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"io"
	"io/ioutil"
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
		fmt.Printf("Init src redis pool [%s] failed.\n", SrcRedis)
		return 2
	}
	DstPool = InitRedisPool(DstRedis)
	if DstPool == nil {
		SrcPool.Close()
		fmt.Printf("Init dst redis pool [%s] failed.\n", DstRedis)
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
		fmt.Printf("exec command [%s] failed.\n", cmdstr)
		return "", errors.New("command failed")
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("StdoutPipe failed,err:%s.\n", err.Error())
		return "", err
	}
	e := cmd.Start()
	if e != nil {
		fmt.Printf("cmd Start failed,err:%s.\n", e.Error())
		return "", e
	}
	bytes, eio := ioutil.ReadAll(stdout)
	if eio != nil {
		fmt.Printf("ReadAll failed,err:%s\n", eio.Error())
		return "", eio
	}
	e = cmd.Wait()
	if e != nil {
		fmt.Printf("Wait failed,err:%s\n", e.Error())
		return "", e
	}
	return string(bytes), nil
}

func KeyFieldMig(src *redis.Conn, dst *redis.Conn, key string, keytype string, field string) error {
	//fmt.Printf("Now migr key [%s]\n", key)
	return nil
}

func Migrate(slotIndex int, key string, keytype string) {
	var cmdstr string
	var tmpfilename string
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
	fmt.Printf("cmd is:[%s]\n", cmdstr)
	_, err := execResult(cmdstr)
	if err != nil {
		fmt.Printf("exec failed,err:%s\n", err.Error())
		return
	}

	f, e := os.Open(tmpfilename)
	if e != nil {
		fmt.Printf("open file [%s] failed,err:%s\n", e.Error())
		return
	}
	defer f.Close()
	SrcConn := SrcPool.Get()
	DstConn := DstPool.Get()
	if SrcConn == nil || DstConn == nil {
		fmt.Printf("Get Conn failed\n")
		return
	}
	defer SrcConn.Close()
	defer DstConn.Close()
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
		KeyFieldMig(&SrcConn, &DstConn, key, keytype, line)
	}
	return
}

func MigrateSlot(filename string, slotIndex int) {
	var slotindex int
	var key string
	var keytype string
	var tmp1, tmp2, tmp3 int

	fp, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Open file [%s] failed.\n", filename)
		return
	}
	defer fp.Close()
	rd := bufio.NewReader(fp)
	for {
		line, err := rd.ReadString('\n')
		if err != nil || io.EOF == err {
			break
		}
		line = strings.Trim(line, "\n")
		n, err := fmt.Sscanf(line, "%d\t%s\t%s\t%d\t%d\t%d\n", &slotindex, &key, &keytype, &tmp1, &tmp2, &tmp3)
		if err != nil || n != 6 {
			fmt.Printf("Sscanf failed,n is:%d,err:%s.\n", n, err.Error())
			continue
		}
		if slotIndex == slotindex {
			Migrate(slotindex, key, keytype)
		}
	}
}

var KeysFile string
var SrcRedis string
var DstRedis string
var SlotId int64

func main() {
	var err error
	c := make(chan os.Signal, 1)
	if len(os.Args) != 5 {
		usage(os.Args[0])
	}
	KeysFile = os.Args[1]
	SrcRedis = os.Args[2]
	DstRedis = os.Args[3]
	SlotId, err = strconv.ParseInt(os.Args[4], 10, 32)
	if err != nil {
		fmt.Printf("invalid slotId:%s\n", os.Args[4])
		usage(os.Args[0])
	}
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		ReleasePool()
		fmt.Printf("recv SIGINTERRUPT,now exit.\n")
		os.Exit(2)
	}()

	if InitAllPool() != 0 {
		fmt.Printf("InitAllPool failed.\n")
		return
	}
	MigrateSlot(KeysFile, int(SlotId))

}
