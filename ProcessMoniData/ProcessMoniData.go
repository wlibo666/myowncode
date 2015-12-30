package main

import (
	"encoding/json"
	"errors"
	"fmt"
	io "io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

type ProxyConf struct {
	ProxyAddr string `json:"addr"`
}

type MoniDataConf struct {
	EmailAddr    string      `json:"email_addr"`
	EmailPwd     string      `json:"email_pwd"`
	MoniInterval int         `json:"moni_interval"`
	ProxyList    []ProxyConf `json:"proxy_list"`
}

func NewConf() *MoniDataConf {
	conf := &MoniDataConf{}
	conf.ProxyList = make([]ProxyConf, 64)
	return conf
}

func LoadConf(fileName string, v interface{}) error {
	data, err := io.ReadFile(fileName)
	if err != nil {
		fmt.Printf("Read file [%s] failed,err:%s\n", fileName, err.Error())
		return err
	}
	//fmt.Printf("json data\n %s \n", string(data))
	err = json.Unmarshal([]byte(data), v)
	if err != nil {
		fmt.Printf("Unmarshal json failed,errr:%s\n", err.Error())
		return err
	}
	//fmt.Printf("struct config:%v\n", v)
	return nil
}

func PrintConf(conf *MoniDataConf) {
	fmt.Printf("email_addr:%s\n", conf.EmailAddr)
	fmt.Printf("email_pwd:%s\n", conf.EmailPwd)
	fmt.Printf("moni_interval:%d\n", conf.MoniInterval)

	for i, proxy := range conf.ProxyList {
		fmt.Printf("  proxy index [%d],addr [%s]\n", i, proxy.ProxyAddr)
	}
	return
}

type PerCmdInfo struct {
	Calls        int64  `json:"calls"`
	Cmd          string `json:"cmd"`
	FailCalls    int64  `json:"fail_calls"`
	FailUsecs    int64  `json:"fail_usecs"`
	Usecs        int64  `json:"usecs"`
	UsecsPerCall int64  `json:"usecs_percall"`
}

type RedisCmdInfo struct {
	RedisAddr string       `json:"RedisAddr"`
	Calls     int64        `json:"Calls"`
	FailCalls int64        `json:"FailCalls"`
	CmdInfo   []PerCmdInfo `json:"Cmdmap"`
}

type RouterInfo struct {
	ProxyCmdInfos []PerCmdInfo   `json:"cmd_proxy"`
	RedisCmdInfos []RedisCmdInfo `json:"cmd_redis"`
	ConnNum       int64          `json:"conn_num"`
	ConnFailNum   int64          `json:"conn_fail_num"`
	OpNum         int64          `json:"op_num"`
	OpFailNum     int64          `json:"op_fail_num"`
	OpSuccNum     int64          `json:"op_succ_num"`
}

type RedisAddrMap struct {
	RedisAddr map[string]bool
}

var AllRedisAddr RedisAddrMap = RedisAddrMap{
	RedisAddr: make(map[string]bool),
}

type MoniData struct {
	ProxyData RouterInfo `json:"router"`
}

func NewMoniData() *MoniData {
	monidata := &MoniData{}
	return monidata
}

func PrintMoniData(proxyAddr string, monidata *MoniData) {
	fmt.Printf("<<<<<<<<<< proxy [%s] >>>>>>>>>\n", proxyAddr)
	fmt.Printf("conn_num:%d\n", monidata.ProxyData.ConnNum)
	fmt.Printf("conn_fail_num:%d\n", monidata.ProxyData.ConnFailNum)
	fmt.Printf("op_num:%d\n", monidata.ProxyData.OpNum)
	fmt.Printf("op_fail_num:%d\n", monidata.ProxyData.OpFailNum)
	fmt.Printf("op_succ_num:%d\n", monidata.ProxyData.OpSuccNum)
	fmt.Printf("------------------------\n")
	for _, proxy := range monidata.ProxyData.ProxyCmdInfos {
		fmt.Printf("cmd:%s\n", proxy.Cmd)
		fmt.Printf("calls:%d\n", proxy.Calls)
		fmt.Printf("fail_calls:%d\n", proxy.FailCalls)
		fmt.Printf("fail_usecs:%d\n", proxy.FailUsecs)
		fmt.Printf("usecs:%d\n", proxy.Usecs)
		fmt.Printf("usecs_percall:%d\n", proxy.UsecsPerCall)
		fmt.Printf("\n")
	}

	fmt.Printf("------------------------\n")
	for _, redis := range monidata.ProxyData.RedisCmdInfos {
		fmt.Printf("RedisAddr:%s\n", redis.RedisAddr)
		fmt.Printf("Calls:%d\n", redis.Calls)
		fmt.Printf("FailCalls:%d\n", redis.FailCalls)
		fmt.Printf("\n")
		for _, cmd := range redis.CmdInfo {
			fmt.Printf("cmd:%s\n", cmd.Cmd)
			fmt.Printf("calls:%d\n", cmd.Calls)
			fmt.Printf("fail_calls:%d\n", cmd.FailCalls)
			fmt.Printf("fail_usecs:%d\n", cmd.FailUsecs)
			fmt.Printf("usecs:%d\n", cmd.Usecs)
			fmt.Printf("usecs_percall:%d\n", cmd.UsecsPerCall)
			fmt.Printf("\n")
		}
		fmt.Printf("------\n")
	}
	fmt.Printf("\n\n")
}

func GetIpAddrByUrl(proxyAddr string) string {
	index := strings.Index(proxyAddr, "//")
	if index == -1 {
		return ""
	}
	index += 2
	end := strings.Index(proxyAddr[index:], ":")
	if end == -1 {
		return ""
	}
	return proxyAddr[index : index+end]
}

func SaveLineFile(fileName string, line string) error {
	return nil
	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, os.ModePerm)
	if err != nil {
		fmt.Printf("openfile [%s] failed.", fileName)
		return err
	}
	defer file.Close()
	file.WriteString(line)
	return nil
}

func GetRedisIpAddr(redisAddr string) string {
	index := strings.Index(redisAddr, ":")
	if index == -1 {
		return ""
	}
	return redisAddr[0:index]
}

func GetTimeStamp() string {
	var t time.Time = time.Now()
	return fmt.Sprintf("%04d-%02d-%02d.%02d:%02d:%02d",
		t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
}

func SaveMoniData(proxyAddr string, monidata *MoniData) error {
	var err error
	// check date
	CheckDate()
	// get proxy addr
	addr := GetIpAddrByUrl(proxyAddr)
	if len(addr) == 0 {
		err = errors.New("get ip addr from [" + proxyAddr + "] failed.")
		fmt.Printf(err.Error())
		return err
	}
	//fmt.Printf("ip addr is :[%s]\n", addr)
	// save proxy data
	timeStr := GetTimeStamp()
	proxyDataFileName := "proxy_" + addr + "." + TodayStr + ".data"
	linedata := fmt.Sprintf("%s\t%d\t%d\t%d\t%d\t%d\n", timeStr,
		monidata.ProxyData.ConnNum, monidata.ProxyData.ConnFailNum,
		monidata.ProxyData.OpNum, monidata.ProxyData.OpFailNum,
		monidata.ProxyData.OpSuccNum)

	SaveLineFile(proxyDataFileName, linedata)
	// save proxy cmd
	proxyCmdFileName := "proxy_" + addr + "." + TodayStr + ".cmd"
	for _, proxy := range monidata.ProxyData.ProxyCmdInfos {
		cmddata := fmt.Sprintf("%s\t%s\t%d\t%d\t%d\t%d\t%d\n", timeStr,
			proxy.Cmd, proxy.Calls, proxy.FailCalls, proxy.FailUsecs,
			proxy.Usecs, proxy.UsecsPerCall)
		SaveLineFile(proxyCmdFileName, cmddata)
	}
	// save redis data
	for _, redis := range monidata.ProxyData.RedisCmdInfos {
		// record all redis addr
		AllRedisAddr.RedisAddr[redis.RedisAddr] = true
		tmpaddr := GetRedisIpAddr(redis.RedisAddr)
		if len(tmpaddr) == 0 {
			tmpaddr = redis.RedisAddr
		}
		redisDataFileName := "redis_" + tmpaddr + "." + TodayStr + ".data"
		redisdata := fmt.Sprintf("%s\t%s\t%d\t%d\n", timeStr, addr, redis.Calls, redis.FailCalls)
		//fmt.Printf("file[%s],data:%s", redisDataFileName, redisdata)
		SaveLineFile(redisDataFileName, redisdata)

		// save redis cmd
		redisCmdFileName := "redis_" + tmpaddr + "." + TodayStr + ".cmd"
		for _, cmd := range redis.CmdInfo {
			redisCmdData := fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\n", timeStr, addr,
				cmd.Cmd, cmd.Calls, cmd.FailCalls, cmd.FailUsecs,
				cmd.Usecs, cmd.UsecsPerCall)
			//fmt.Printf("file[%s],cmd:%s", redisCmdFileName, redisCmdData)
			SaveLineFile(redisCmdFileName, redisCmdData)
		}
	}
	return nil
}

func CollectMoniData(conf *MoniDataConf) error {
	for {
		for _, proxy := range conf.ProxyList {
			resp, err := http.Get(proxy.ProxyAddr)
			defer resp.Body.Close()
			if err != nil {
				fmt.Printf("Get [%s] failed,err:%s\n", proxy.ProxyAddr, err.Error())
				continue
			}
			data, err := io.ReadAll(resp.Body)
			//fmt.Printf("data:%s\n", string(data))
			monidata := NewMoniData()
			err = json.Unmarshal([]byte(data), &monidata)
			if err != nil {
				fmt.Printf("unmarshal data from add [%s] failed,err:%s.\n", proxy.ProxyAddr, err.Error())
				continue
			}
			//PrintMoniData(proxy.ProxyAddr, monidata)

			err = SaveMoniData(proxy.ProxyAddr, monidata)
			if err != nil {
				continue
			}
		}
		for i := 0; i < conf.MoniInterval; i++ {
			time.Sleep(time.Second)
		}
	}
	return nil
}

func usage(progName string) {
	fmt.Printf("Usage:%s or %s configFile\n", progName, progName)
	os.Exit(0)
}

func GetTimeStr(t time.Time) string {
	return fmt.Sprintf("%04d%02d%02d", t.Year(), t.Month(), t.Day())
}

func SendDayReport(conf *MoniDataConf) error {
	for _, proxy := range conf.ProxyList {
		//fmt.Printf("proxy addr:%s\n", proxy.ProxyAddr)
		addr := GetIpAddrByUrl(proxy.ProxyAddr)
		if len(addr) == 0 {
			fmt.Printf("get ip addr from [%s] failed", proxy.ProxyAddr)
			continue
		}
		// Get Proxy day data
		proxyDataFileName := "proxy_" + addr + "." + PreTimeFalg + ".data"
		proxyDataNode := GetProxyDayData(proxyDataFileName, conf.MoniInterval)
		if proxyDataNode == nil {
			continue
		}
		/*fmt.Printf("interval:%d\n", proxyDataNode.TimeInterval)
		fmt.Printf("ConnNum:%d\n", proxyDataNode.ConnNum)
		fmt.Printf("ConnFailNum:%d\n", proxyDataNode.ConnFailNum)
		fmt.Printf("OpNum:%d\n", proxyDataNode.OpNum)
		fmt.Printf("OpFailNum:%d\n", proxyDataNode.OpFailNum)
		fmt.Printf("OpSuccNum:%d\n", proxyDataNode.OpSuccNum)*/

		// Get Proxy day cmd
		proxyCmdFileName := "proxy_" + addr + "." + PreTimeFalg + ".cmd"
		proxyCmdMap := GetProxyDayCmd(proxyCmdFileName, conf.MoniInterval)
		if proxyCmdMap == nil {
			continue
		}
		//fmt.Printf("Now Print CmdMap\n")
		//PrintProxyCmdMap(proxyCmdMap)
	}

	for redisAddr, _ := range AllRedisAddr.RedisAddr {
		tmpaddr := GetRedisIpAddr(redisAddr)
		if len(tmpaddr) == 0 {
			tmpaddr = redisAddr
		}
		// Get redis day data
		redisDataFileName := "redis_" + tmpaddr + "." + PreTimeFalg + ".data"
		redisDataMap := GetRedisDayData(redisDataFileName, conf.MoniInterval)
		if redisDataMap == nil {
			continue
		}
		//PrintRedisDataMap(redisDataMap)

		// Get redis day cmd
		redisCmdFileName := "redis_" + tmpaddr + "." + PreTimeFalg + ".cmd"
		redisCmdMap := GetRedisDayCmd(redisCmdFileName, conf.MoniInterval)
		if redisCmdMap == nil {
			continue
		}
		fmt.Printf("redis addr[%s]\n", redisAddr)
		PrintRedisCmdMap(redisCmdMap)
	}
	return nil
}

var PreTimeFalg string = ""

func CheckDate() {
	var NowTime time.Time = time.Now()
	var tmpStr string = GetTimeStr(NowTime)

	if TodayStr != tmpStr {
		PreTimeFalg = tmpStr
		go func() {
			err := SendDayReport(GlobalConfig)
			if err != nil {
				fmt.Printf("send report failed,err:%s\n", err.Error())
			}
		}()
		TodayStr = tmpStr
	}
}

var ConfFile string = "monidata.json"
var GlobalConfig *MoniDataConf = NewConf()
var TodayTime time.Time = time.Now()
var TodayStr string = GetTimeStr(TodayTime)

func TestSendReport() {
	for i := 0; i < 20; i++ {
		time.Sleep(time.Second)
	}
	PreTimeFalg = "20151230"
	SendDayReport(GlobalConfig)
}

func main() {
	var argNum int = len(os.Args)
	if argNum != 1 && argNum != 2 {
		usage(os.Args[0])
	}
	if argNum == 2 {
		ConfFile = os.Args[1]
	}
	// Load config
	err := LoadConf(ConfFile, &GlobalConfig)
	if err != nil {
		os.Exit(2)
	}
	//PrintConf(GlobalConfig)
	// get monitor data
	go func() {
		err := CollectMoniData(GlobalConfig)
		if err != nil {
			fmt.Printf("CollectMoniData failed,err:%s\n", err.Error())
			os.Exit(2)
		}
	}()
	TestSendReport()
	for {
		time.Sleep(time.Second)
	}
}
