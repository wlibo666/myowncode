package main

import (
	"encoding/json"
	"errors"
	"fmt"
	io "io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type ProxyConf struct {
	ProxyAddr string `json:"addr"`
}

type MoniDataConf struct {
	EmailAddr    string      `json:"email_addr"`
	EmailPwd     string      `json:"email_pwd"`
	SmtpAddr     string      `json:"smtp_addr"`
	ToAddr       string      `json:"to_addr"`
	SendTime     string      `json:"send_time"`
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
		GLogger.Printf("Read file [%s] failed,err:%s\n", fileName, err.Error())
		return err
	}

	//GLogger.Printf("json data\n %s \n", string(data))
	err = json.Unmarshal([]byte(data), v)
	if err != nil {
		GLogger.Printf("Unmarshal json failed,errr:%s\n", err.Error())
		return err
	}
	//GLogger.Printf("struct config:%v\n", v)
	return nil
}

func PrintConf(conf *MoniDataConf) {
	GLogger.Printf("email_addr:%s\n", conf.EmailAddr)
	GLogger.Printf("email_pwd:%s\n", conf.EmailPwd)
	GLogger.Printf("smtp_addr:%s\n", conf.SmtpAddr)
	GLogger.Printf("to_addr:%s\n", conf.ToAddr)
	GLogger.Printf("send_time:%s\n", conf.SendTime)
	GLogger.Printf("moni_interval:%d\n", conf.MoniInterval)

	for i, proxy := range conf.ProxyList {
		GLogger.Printf("  proxy index [%d],addr [%s]\n", i, proxy.ProxyAddr)
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
	GLogger.Printf("<<<<<<<<<< proxy [%s] >>>>>>>>>\n", proxyAddr)
	GLogger.Printf("conn_num:%d\n", monidata.ProxyData.ConnNum)
	GLogger.Printf("conn_fail_num:%d\n", monidata.ProxyData.ConnFailNum)
	GLogger.Printf("op_num:%d\n", monidata.ProxyData.OpNum)
	GLogger.Printf("op_fail_num:%d\n", monidata.ProxyData.OpFailNum)
	GLogger.Printf("op_succ_num:%d\n", monidata.ProxyData.OpSuccNum)
	GLogger.Printf("------------------------\n")
	for _, proxy := range monidata.ProxyData.ProxyCmdInfos {
		GLogger.Printf("cmd:%s\n", proxy.Cmd)
		GLogger.Printf("calls:%d\n", proxy.Calls)
		GLogger.Printf("fail_calls:%d\n", proxy.FailCalls)
		GLogger.Printf("fail_usecs:%d\n", proxy.FailUsecs)
		GLogger.Printf("usecs:%d\n", proxy.Usecs)
		GLogger.Printf("usecs_percall:%d\n", proxy.UsecsPerCall)
		GLogger.Printf("\n")
	}

	GLogger.Printf("------------------------\n")
	for _, redis := range monidata.ProxyData.RedisCmdInfos {
		GLogger.Printf("RedisAddr:%s\n", redis.RedisAddr)
		GLogger.Printf("Calls:%d\n", redis.Calls)
		GLogger.Printf("FailCalls:%d\n", redis.FailCalls)
		GLogger.Printf("\n")
		for _, cmd := range redis.CmdInfo {
			GLogger.Printf("cmd:%s\n", cmd.Cmd)
			GLogger.Printf("calls:%d\n", cmd.Calls)
			GLogger.Printf("fail_calls:%d\n", cmd.FailCalls)
			GLogger.Printf("fail_usecs:%d\n", cmd.FailUsecs)
			GLogger.Printf("usecs:%d\n", cmd.Usecs)
			GLogger.Printf("usecs_percall:%d\n", cmd.UsecsPerCall)
			GLogger.Printf("\n")
		}
		GLogger.Printf("------\n")
	}
	GLogger.Printf("\n\n")
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
	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, os.ModePerm)
	if err != nil {
		GLogger.Printf("openfile [%s] failed.", fileName)
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
		GLogger.Printf(err.Error())
		return err
	}
	//GLogger.Printf("ip addr is :[%s]\n", addr)
	// save proxy data
	timeStr := GetTimeStamp()
	proxyDataFileName := ".proxy_" + addr + "." + TodayStr + ".data"
	linedata := fmt.Sprintf("%s\t%d\t%d\t%d\t%d\t%d\n", timeStr,
		monidata.ProxyData.ConnNum, monidata.ProxyData.ConnFailNum,
		monidata.ProxyData.OpNum, monidata.ProxyData.OpFailNum,
		monidata.ProxyData.OpSuccNum)

	SaveLineFile(proxyDataFileName, linedata)
	// save proxy cmd
	proxyCmdFileName := ".proxy_" + addr + "." + TodayStr + ".cmd"
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
		redisDataFileName := ".redis_" + tmpaddr + "." + TodayStr + ".data"
		redisdata := fmt.Sprintf("%s\t%s\t%d\t%d\n", timeStr, addr, redis.Calls, redis.FailCalls)
		//GLogger.Printf("file[%s],data:%s", redisDataFileName, redisdata)
		SaveLineFile(redisDataFileName, redisdata)

		// save redis cmd
		redisCmdFileName := ".redis_" + tmpaddr + "." + TodayStr + ".cmd"
		for _, cmd := range redis.CmdInfo {
			redisCmdData := fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\n", timeStr, addr,
				cmd.Cmd, cmd.Calls, cmd.FailCalls, cmd.FailUsecs,
				cmd.Usecs, cmd.UsecsPerCall)
			//GLogger.Printf("file[%s],cmd:%s", redisCmdFileName, redisCmdData)
			SaveLineFile(redisCmdFileName, redisCmdData)
		}
	}
	return nil
}

func CollectMoniData(conf *MoniDataConf) error {
	for {
		for _, proxy := range conf.ProxyList {
			resp, err := http.Get(proxy.ProxyAddr)
			if err != nil {
				GLogger.Printf("Get [%s] failed,err:%s\n", proxy.ProxyAddr, err.Error())
				continue
			}

			if resp == nil || resp.Body == nil {
				GLogger.Printf("conn get [%s] failed\n", proxy.ProxyAddr)
				continue
			}
			defer resp.Body.Close()
			data, err := io.ReadAll(resp.Body)
			//GLogger.Printf("data:%s\n", string(data))
			monidata := NewMoniData()
			err = json.Unmarshal([]byte(data), &monidata)
			if err != nil {
				GLogger.Printf("unmarshal data from add [%s] failed,err:%s.\n", proxy.ProxyAddr, err.Error())
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
	var proxyData string = ""
	//var proxyCmd string = ""
	var redisData string = ""
	var redisCmd string = ""
	for _, proxy := range conf.ProxyList {
		//GLogger.Printf("proxy addr:%s\n", proxy.ProxyAddr)
		addr := GetIpAddrByUrl(proxy.ProxyAddr)
		if len(addr) == 0 {
			GLogger.Printf("get ip addr from [%s] failed", proxy.ProxyAddr)
			continue
		}
		// Get Proxy day data
		proxyDataFileName := ".proxy_" + addr + "." + PreTimeFalg + ".data"
		proxyDataNode := GetProxyDayData(proxyDataFileName, conf.MoniInterval)
		if proxyDataNode == nil {
			continue
		}

		proxyData += GenProxyDataHtml(addr, proxyDataNode)
		/*GLogger.Printf("interval:%d\n", proxyDataNode.TimeInterval)
		GLogger.Printf("ConnNum:%d\n", proxyDataNode.ConnNum)
		GLogger.Printf("ConnFailNum:%d\n", proxyDataNode.ConnFailNum)
		GLogger.Printf("OpNum:%d\n", proxyDataNode.OpNum)
		GLogger.Printf("OpFailNum:%d\n", proxyDataNode.OpFailNum)
		GLogger.Printf("OpSuccNum:%d\n", proxyDataNode.OpSuccNum)*/

		// Get Proxy day cmd
		/*proxyCmdFileName := ".proxy_" + addr + "." + PreTimeFalg + ".cmd"
		proxyCmdMap := GetProxyDayCmd(proxyCmdFileName, conf.MoniInterval)
		if proxyCmdMap == nil {
			continue
		}
		proxyCmd += GenProxyCmdHtml(addr, proxyCmdMap)*/
		//GLogger.Printf("Now Print CmdMap\n")
		//PrintProxyCmdMap(proxyCmdMap)
	}
	InitAllRedisCmdNode()
	for redisAddr, _ := range AllRedisAddr.RedisAddr {
		tmpaddr := GetRedisIpAddr(redisAddr)
		if len(tmpaddr) == 0 {
			tmpaddr = redisAddr
		}
		// Get redis day data
		redisDataFileName := ".redis_" + tmpaddr + "." + PreTimeFalg + ".data"
		redisDataMap := GetRedisDayData(redisDataFileName, conf.MoniInterval)
		if redisDataMap == nil {
			continue
		}

		redisData += GenRedisSummaryHtml(tmpaddr, redisDataMap)
		//PrintRedisDataMap(redisDataMap)

		// Get redis day cmd
		redisCmdFileName := ".redis_" + tmpaddr + "." + PreTimeFalg + ".cmd"
		redisCmdMap := GetRedisDayCmd(redisCmdFileName, conf.MoniInterval)
		if redisCmdMap == nil {
			continue
		}
		CalcRedisCmd(tmpaddr, redisCmdMap)
		//GLogger.Printf("redis addr[%s]\n", redisAddr)
		//PrintRedisCmdMap(redisCmdMap)
	}
	redisCmd += GenRedisCmdHtml()
	//GLogger.Printf("proxyData:%s\n", proxyData)
	//GLogger.Printf("proxyCmd:%s\n", proxyCmd)
	//GLogger.Print("redisData:%s\n", redisData)
	//GLogger.Print("redisCmd:%s\n", redisCmd)
	html := GenDaySummaryReportHtml(proxyData, redisData, redisCmd)
	var Subject string = "Codis集群监控统计 (" + PreTimeFalg + ")"
	GLogger.Printf("report content:\n%s\n", html)
	err := SendSmtpEmail(GlobalConfig.EmailAddr, conf.EmailPwd, conf.SmtpAddr, conf.ToAddr, Subject, html, "html")
	if err != nil {
		GLogger.Printf("Send Email [%s] to [%s] failed,err:%s\n", Subject, GlobalConfig.ToAddr, err.Error())
	} else {
		GLogger.Printf("Send Email [%s] to [%s] success\n", Subject, GlobalConfig.ToAddr)
	}
	return nil
}

var PreTimeFalg string = ""

func CheckDate() {
	var NowTime time.Time = time.Now()
	var tmpStr string = GetTimeStr(NowTime)

	if TodayStr != tmpStr {
		PreTimeFalg = TodayStr
		write_date(PreTimeFalg)
		go func() {
			for {
				var t time.Time = time.Now()
				if strconv.FormatInt(int64(t.Hour()), 10) != GlobalConfig.SendTime {
					time.Sleep(time.Second * 60)
					continue
				}
				break
			}
			err := SendDayReport(GlobalConfig)
			if err != nil {
				GLogger.Printf("send report failed,err:%s\n", err.Error())
			}
		}()
		TodayStr = tmpStr
	}
}

var LogFileName = "MoniCoids.log"
var GLogger *log.Logger
var ConfFile string = "monidata.json"
var GlobalConfig *MoniDataConf = NewConf()
var TodayTime time.Time = time.Now()
var TodayStr string = GetTimeStr(TodayTime)

func TestSendReport() {
	for i := 0; i < 300; i++ {
		time.Sleep(time.Second)
	}
	PreTimeFalg = "20160106"
	SendDayReport(GlobalConfig)
}

func InitLog() {
	file, err := os.OpenFile(LogFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, os.ModePerm)
	if err != nil {
		fmt.Printf("Create log file [%s] failed,err:%s\n", LogFileName, err.Error())
		os.Exit(2)
	}
	GLogger = log.New(file, "", log.LstdFlags)
}

func DeleteExpireFile(days int) {
	now := time.Now().Unix()
	var daySecs int64 = int64(days * 24 * 3600)
	filepath.Walk("./", func(path string, f os.FileInfo, err error) error {
		if strings.Contains(path, ".proxy_") || strings.Contains(path, ".redis_") {
			if (now - f.ModTime().Unix()) > daySecs {
				GLogger.Printf("now delete file [%s]", path)
				os.Remove(path)
			}
		}
		return nil
	})
}

func write_date(date string) {
	f, err := os.OpenFile(".date", os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(date)
	GLogger.Printf("write date [%s]", date)
}

func read_date() string {
	f, err := os.Open(".date")
	if err != nil {
		return ""
	}
	defer f.Close()
	var b []byte = make([]byte, 8)
	f.Read(b)
	return string(b)
}

func main() {
	var argNum int = len(os.Args)
	if argNum != 1 && argNum != 2 {
		usage(os.Args[0])
	}
	LogFileName = os.Args[0] + ".log"
	if argNum == 2 {
		ConfFile = os.Args[1]
	}
	InitLog()
	PreTimeFalg = read_date()
	GLogger.Printf("get pre date=%s", PreTimeFalg)
	// Load config
	err := LoadConf(ConfFile, &GlobalConfig)
	if err != nil {
		os.Exit(2)
	}
	GLogger.Printf("[%s] start.", os.Args[0])
	//PrintConf(GlobalConfig)
	// get monitor data
	go func() {
		err := CollectMoniData(GlobalConfig)
		if err != nil {
			GLogger.Printf("CollectMoniData failed,err:%s\n", err.Error())
			os.Exit(2)
		}
	}()
	//TestSendReport()
	for {
		time.Sleep(time.Second * 3600)
		DeleteExpireFile(2)
	}
}
