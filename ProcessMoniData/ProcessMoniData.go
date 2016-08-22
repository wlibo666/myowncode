package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
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
	WarnAddr     string      `json:"warn_addr"`
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
	data, err := ioutil.ReadFile(fileName)
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
	GLogger.Printf("warn_addr:%s\n", conf.WarnAddr)
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
	monidata.ProxyData.ProxyCmdInfos = make([]PerCmdInfo, 0, 64)
	monidata.ProxyData.RedisCmdInfos = make([]RedisCmdInfo, 0, 256)
	return monidata
}

func MoniDataInit(data *MoniData) {
	for _, cmd := range data.ProxyData.ProxyCmdInfos {
		cmd.Calls = 0
		cmd.Cmd = ""
		cmd.FailCalls = 0
		cmd.FailUsecs = 0
		cmd.Usecs = 0
		cmd.UsecsPerCall = 0
	}
	/*for _, redis := range ProxyData.RedisCmdInfos {
		redis.
	}*/
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
		if len(redis.RedisAddr) == 0 {
			continue
		}
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

// 上一次的代理信息
var PreStatisticProxyData [64]*ProxyPerDataNode

// 当天的代理统计信息
var CurrProxyData [64]*ProxyPerDataNode

func CollectMoniData(conf *MoniDataConf) error {
	var index int = 0
	var data []byte
	var resp *http.Response
	var err error
	monidata := NewMoniData()
	for {
		index = 0

		//GLogger.Printf("now will get monidata")
		for _, proxy := range conf.ProxyList {
			resp, err = http.Get(proxy.ProxyAddr)
			if err != nil {
				GLogger.Printf("Get [%s] failed,err:%s\n", proxy.ProxyAddr, err.Error())
				continue
			}

			if resp == nil || resp.Body == nil {
				GLogger.Printf("conn get [%s] failed\n", proxy.ProxyAddr)
				continue
			}
			defer resp.Body.Close()
			data, err = ioutil.ReadAll(resp.Body)
			//GLogger.Printf("data:%s\n", string(data))
			err = json.Unmarshal([]byte(data), &monidata)
			if err != nil {
				GLogger.Printf("unmarshal data from add [%s] failed,err:%s.\n", proxy.ProxyAddr, err.Error())
				continue
			}

			//GLogger.Printf("---print moni data----")
			//PrintMoniData(proxy.ProxyAddr, monidata)

			err = SaveMoniData(proxy.ProxyAddr, monidata)
			if err != nil {
				continue
			}

			//GLogger.Printf("------print pre proxy data-----")
			//PrintCurrProxyData(PreStatisticProxyData)

			//GLogger.Printf("----print moni proxy data----")
			//GLogger.Printf("now conn:%d,failConn:%d,OpNum:%d,succOp:%d", monidata.ProxyData.ConnNum, monidata.ProxyData.ConnFailNum, monidata.ProxyData.OpNum, monidata.ProxyData.OpSuccNum)
			// 记录当天的统计结果
			tmpCur := CurrProxyData[index]
			if tmpCur == nil {
				//GLogger.Printf("currproxydata[%d] is nil", index)
				tmpCur = &ProxyPerDataNode{
					Addr:      proxy.ProxyAddr,
					StartTime: fmt.Sprintf("%d", time.Now().Unix()),
					EndTime:   fmt.Sprintf("%d", time.Now().Unix()),
				}
				CurrProxyData[index] = tmpCur
			} else {
				tmpCur.EndTime = fmt.Sprintf("%d", time.Now().Unix())
			}

			// 保存上一次代理统计记录
			tmp := PreStatisticProxyData[index]
			if tmp == nil {
				//GLogger.Printf("Prestaticsticproxydata[%d] is nil", index)
				tmp = &ProxyPerDataNode{
					Addr:        proxy.ProxyAddr,
					StartTime:   fmt.Sprintf("%d", time.Now().Unix()),
					EndTime:     fmt.Sprintf("%d", time.Now().Unix()),
					ConnNum:     monidata.ProxyData.ConnNum,
					ConnFailNum: monidata.ProxyData.ConnFailNum,
					OpNum:       monidata.ProxyData.OpNum,
					OpFailNum:   monidata.ProxyData.OpFailNum,
					OpSuccNum:   monidata.ProxyData.OpSuccNum,
				}
				PreStatisticProxyData[index] = tmp
				// 没保存上一次初值，不累加
			} else {
				// 有上一次值,计算差值
				//GLogger.Printf("now conn:%d,failConn:%d,OpNum:%d,succOp:%d", monidata.ProxyData.ConnNum, monidata.ProxyData.ConnFailNum, monidata.ProxyData.OpNum, monidata.ProxyData.OpSuccNum)
				//GLogger.Printf("pre conn:%d,failConn:%d,OpNum:%d,succop:%d", tmp.ConnNum, tmp.ConnFailNum, tmp.OpNum, tmp.OpSuccNum)
				if monidata.ProxyData.ConnNum >= tmp.ConnNum {
					tmpCur.ConnNum += (monidata.ProxyData.ConnNum - tmp.ConnNum)
				} else {
					tmpCur.ConnNum += monidata.ProxyData.ConnNum
				}
				if monidata.ProxyData.ConnFailNum >= tmp.ConnFailNum {
					tmpCur.ConnFailNum += (monidata.ProxyData.ConnFailNum - tmp.ConnFailNum)
				} else {
					tmpCur.OpNum += monidata.ProxyData.ConnFailNum
				}
				if monidata.ProxyData.OpNum >= tmp.OpNum {
					tmpCur.OpNum += (monidata.ProxyData.OpNum - tmp.OpNum)
				} else {
					tmpCur.OpNum += monidata.ProxyData.OpNum
				}
				if monidata.ProxyData.OpFailNum >= tmp.OpFailNum {
					tmpCur.OpFailNum += (monidata.ProxyData.OpFailNum - tmp.OpFailNum)
				} else {
					tmpCur.OpFailNum += monidata.ProxyData.OpFailNum
				}
				if monidata.ProxyData.OpSuccNum >= tmp.OpSuccNum {
					tmpCur.OpSuccNum += (monidata.ProxyData.OpSuccNum - tmp.OpSuccNum)
				} else {
					tmpCur.OpSuccNum += monidata.ProxyData.OpSuccNum
				}
				// 将当前值覆盖上一次值
				tmp.ConnNum = monidata.ProxyData.ConnNum
				tmp.ConnFailNum = monidata.ProxyData.ConnFailNum
				tmp.OpNum = monidata.ProxyData.OpNum
				tmp.OpFailNum = monidata.ProxyData.OpFailNum
				tmp.OpSuccNum = monidata.ProxyData.OpSuccNum
			}

			// 处理redis数据
			ProcessRedisMoniData(index, proxy.ProxyAddr, monidata)
			index++
		}
		// 将最新的统计结果持久化到文件
		SaveCurProxyRecord()
		// 将上一次值持久化到文件
		SavePreProxyRecord()
		// 将最新的redis数据统计结果持久化到文件
		SaveCurRedisRecord()
		// 将上一次的统计值持久化到文件
		SavePreRedisRecord()

		//GLogger.Printf("print curr proxy data after get monidata")
		//PrintCurrProxyData(CurrProxyData)

		// 根据统计数据检测服务器是否出现问题，如果有则处理
		CheckServerAndWarn()
		for i := 0; i < conf.MoniInterval; i++ {
			time.Sleep(time.Second)
		}
	}
	return nil
}

var PreEmailTime int64 = 0
var PreProxyConnFailNum [64]int64
var PreProxyOpFailNum [64]int64

func CheckServerAndWarn() {
	var datastr string = ""
	for index, proxy := range CurrProxyData {
		if proxy == nil {
			continue
		}
		tmpaddr := GetIpAddrByUrl(proxy.Addr)
		if len(tmpaddr) == 0 {
			tmpaddr = proxy.Addr
		}
		// 连接数失败过多
		if float64(proxy.ConnFailNum)/float64(proxy.ConnNum) > 0.01 {
			if PreProxyConnFailNum[index] < proxy.ConnFailNum {
				s := fmt.Sprintf("proxy[%s] connection number [%d] but fail number [%d],please check it.{new-line}",
					tmpaddr, proxy.ConnNum, proxy.ConnFailNum)
				datastr += s
				PreProxyConnFailNum[index] = proxy.ConnFailNum
			}
		}
		// 操作失败过多
		if float64(proxy.OpFailNum)/float64(proxy.OpNum) > 0.0002 {
			if PreProxyOpFailNum[index] < proxy.OpFailNum {
				s := fmt.Sprintf("proxy[%s] op number [%d] but fail number [%d],please check it.{new-line}",
					tmpaddr, proxy.OpNum, proxy.OpFailNum)
				datastr += s
				PreProxyOpFailNum[index] = proxy.OpFailNum
			}
		}
	}
	/*for addr, record := range CurrRedisData.Records {
		if float64(record.OpFailNum)/float64(record.OpNum) > 0.0002 {
			s := fmt.Sprintf("redis [%s] op number [%d],but fail num [%d],please check it.{new-line}",
				addr, record.OpNum, record.OpFailNum)
			datastr += s
		}
	}*/

	if len(datastr) <= 10 {
		return
	}
	datastr = strings.Replace(datastr, "{new-line}", "\n", -1)
	GLogger.Printf("warn msg:[%s]", datastr)

	tt := time.Now()
	Subject := fmt.Sprintf("codis集群告警(%04d-%02d-%02d %02d:%02d:%02d)", tt.Year(), tt.Month(), tt.Day(),
		tt.Hour(), tt.Minute(), tt.Second())

	t := tt.Unix()

	if (PreEmailTime != 0) && (t-PreEmailTime) < 300 {
		GLogger.Printf("something error but now time is too short from last sending warn email,should not send now.")
		return
	}
	for _, addr := range strings.Split(GlobalConfig.WarnAddr, ";") {
		err := SendSmtpEmail(GlobalConfig.EmailAddr, GlobalConfig.EmailPwd, GlobalConfig.SmtpAddr, addr, Subject, datastr, "text")
		if err != nil {
			GLogger.Printf("Send Warn Email [%s] to [%s] failed,err:%s\n", Subject, addr, err.Error())
		} else {
			GLogger.Printf("Send Warn Email [%s] to [%s] success\n", Subject, addr)
			PreEmailTime = t
		}
	}
}

// 关于redis操作
type RedisDataRecord struct {
	StartTime string
	EndTime   string
	OpNum     int64
	OpFailNum int64
}

type RedisDataStatistic struct {
	ProxyAddr string
	Records   map[string]*RedisDataRecord
}

var CurrRedisData RedisDataStatistic = RedisDataStatistic{
	Records: make(map[string]*RedisDataRecord),
}

var StatisPreRedisData [64]*RedisDataStatistic

type RedisCmdRecord struct {
	StartTime  string
	EndTime    string
	OpNum      int64
	OpFailNum  int64
	OpSecs     int64
	OpFailSecs int64
}

type RedisCmdStatistic struct {
	ProxyAddr string
	Cmds      map[string]*RedisCmdRecord
}

var CurrRedisCmd RedisCmdStatistic = RedisCmdStatistic{
	Cmds: make(map[string]*RedisCmdRecord),
}

var StatisPreRedisCmd [64]*RedisCmdStatistic

func PrintRedisData(data *RedisDataStatistic) {
	if data == nil {
		return
	}
	for addr, record := range data.Records {
		GLogger.Printf("addr[%s],start[%s],end[%s],opnum[%d],opfailnum[%d]", addr, record.StartTime,
			record.EndTime, record.OpNum, record.OpFailNum)
	}
}

func PrintRedisCmd(cmds *RedisCmdStatistic) {
	if cmds == nil {
		return
	}
	for cmdname, cmdinfo := range cmds.Cmds {
		GLogger.Printf("cmd[%s],start[%s],end[%s],opnum[%d],opsec[%d],opfailnum[%d],opfailsec[%d]",
			cmdname, cmdinfo.StartTime, cmdinfo.EndTime, cmdinfo.OpNum, cmdinfo.OpSecs,
			cmdinfo.OpFailNum, cmdinfo.OpFailSecs)
	}
}

func GenPreRedisDataName() string {
	return fmt.Sprintf(".pre_redis_data.%s", PreTimeFlag)
}

func GenPreRedisCmdName() string {
	return fmt.Sprintf(".pre_redis_cmd.%s", PreTimeFlag)
}

func GenNowPreRedisDataName() string {
	return fmt.Sprintf(".pre_redis_data.%s", TodayStr)
}

func GenNowPreRedisCmdName() string {
	return fmt.Sprintf(".pre_redis_cmd.%s", TodayStr)
}

func SaveCurRedisData(filename string, data *RedisDataStatistic) {
	tmpfile, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if tmpfile == nil {
		GLogger.Printf("open file [%s] failed,err:%s", filename, err.Error())
	} else {
		defer tmpfile.Close()

		for addr, record := range data.Records {
			s := fmt.Sprintf("%s\t%s\t%s\t%d\t%d\n", addr, record.StartTime,
				record.EndTime, record.OpNum, record.OpFailNum)
			tmpfile.WriteString(s)
			//GLogger.Printf("cur proxy data %s [%s]", time.Now().String(), s)
		}
	}
}

func SaveRedisStatisticData(filename string, alldata [64]*RedisDataStatistic) {
	tmpfile, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if tmpfile == nil {
		GLogger.Printf("open file [%s] failed,err:%s", filename, err.Error())
	} else {
		defer tmpfile.Close()
		for _, data := range alldata {
			if data == nil {
				continue
			}
			for addr, record := range data.Records {
				s := fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%d\n", data.ProxyAddr, addr, record.StartTime,
					record.EndTime, record.OpNum, record.OpFailNum)
				tmpfile.WriteString(s)
				//GLogger.Printf("cur proxy data %s [%s]", time.Now().String(), s)
			}
		}
	}
}

func SaveCurRedisCmd(filename string, cmds *RedisCmdStatistic) {
	tmpfile, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if tmpfile == nil {
		GLogger.Printf("open file [%s] failed,err:%s", filename, err.Error())
	} else {
		defer tmpfile.Close()

		for cmdname, cmdinfo := range cmds.Cmds {
			if cmdinfo.OpNum <= 0 {
				continue
			}
			s := fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%d\n", cmdname, cmdinfo.StartTime,
				cmdinfo.EndTime, cmdinfo.OpNum, cmdinfo.OpSecs, cmdinfo.OpFailNum,
				cmdinfo.OpFailSecs)
			tmpfile.WriteString(s)
			//GLogger.Printf("cur proxy data %s [%s]", time.Now().String(), s)
		}
	}
}

func SaveRedisStatisticCmd(filename string, allcmds [64]*RedisCmdStatistic) {
	tmpfile, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if tmpfile == nil {
		GLogger.Printf("open file [%s] failed,err:%s", filename, err.Error())
	} else {
		defer tmpfile.Close()
		for _, cmds := range allcmds {
			if cmds == nil {
				continue
			}
			for cmdname, cmdinfo := range cmds.Cmds {
				if cmdinfo.OpNum <= 0 {
					continue
				}
				s := fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%d\t%d\t%d\n", cmds.ProxyAddr, cmdname, cmdinfo.StartTime,
					cmdinfo.EndTime, cmdinfo.OpNum, cmdinfo.OpSecs, cmdinfo.OpFailNum,
					cmdinfo.OpFailSecs)
				tmpfile.WriteString(s)
				//GLogger.Printf("cur proxy data %s [%s]", time.Now().String(), s)
			}
		}
	}
}

func ProcessRedisMoniData(index int, proxyaddr string, monidata *MoniData) {
	if monidata == nil {
		return
	}
	// 临时统计data变量
	var TmpRedisData *RedisDataStatistic = &RedisDataStatistic{
		ProxyAddr: proxyaddr,
		Records:   make(map[string]*RedisDataRecord),
	}
	// 临时统计cmd变量
	var TmpRedisCmd *RedisCmdStatistic = &RedisCmdStatistic{
		ProxyAddr: proxyaddr,
		Cmds:      make(map[string]*RedisCmdRecord),
	}

	// 计算本次临时统计数据
	for _, redisinfo := range monidata.ProxyData.RedisCmdInfos {
		if len(redisinfo.RedisAddr) == 0 {
			continue
		}
		// 找到临时统计中的redis
		tmpredis := TmpRedisData.Records[redisinfo.RedisAddr]
		if tmpredis == nil {
			tmpredis = &RedisDataRecord{
				StartTime: fmt.Sprintf("%d", time.Now().Unix()),
				EndTime:   fmt.Sprintf("%d", time.Now().Unix()),
			}
			TmpRedisData.Records[redisinfo.RedisAddr] = tmpredis
		} else {
			tmpredis.EndTime = fmt.Sprintf("%d", time.Now().Unix())
		}
		for _, cmdinfo := range redisinfo.CmdInfo {
			// 临时统计单个redis操作
			tmpredis.OpNum += cmdinfo.Calls
			tmpredis.OpFailNum += cmdinfo.FailCalls
			// 找到临时统计中的cmd
			tmpcmd := TmpRedisCmd.Cmds[cmdinfo.Cmd]
			if tmpcmd == nil {
				tmpcmd = &RedisCmdRecord{
					StartTime: fmt.Sprintf("%d", time.Now().Unix()),
					EndTime:   fmt.Sprintf("%d", time.Now().Unix()),
				}
				TmpRedisCmd.Cmds[cmdinfo.Cmd] = tmpcmd
			} else {
				tmpcmd.EndTime = fmt.Sprintf("%d", time.Now().Unix())
			}
			tmpcmd.OpNum += cmdinfo.Calls
			tmpcmd.OpFailNum += cmdinfo.FailCalls
			tmpcmd.OpSecs += cmdinfo.Usecs
			tmpcmd.OpFailSecs += cmdinfo.FailUsecs
		}
	}

	PreRedisData := StatisPreRedisData[index]
	PreRedisCmd := StatisPreRedisCmd[index]
	// 打印临时数据
	//GLogger.Printf("----Print pre redis data---")
	//PrintRedisData(PreRedisData)
	//GLogger.Printf("----Print tmp redis data---")
	//PrintRedisData(TmpRedisData)
	//GLogger.Printf("----Print pre redis cmd, index [%d]----", index)
	//PrintRedisCmd(PreRedisCmd)
	//GLogger.Printf("----Print tmp redis cmd----")
	//PrintRedisCmd(TmpRedisCmd)

	// 根据临时数据差值更新当前统计(data数据)
	for addr, record := range TmpRedisData.Records {
		if PreRedisData == nil {
			tmpCur := CurrRedisData.Records[addr]
			if tmpCur == nil {
				// 且当前也没记录，添加新记录，但计数忽略
				tmpCur = &RedisDataRecord{
					StartTime: fmt.Sprintf("%d", time.Now().Unix()),
					EndTime:   fmt.Sprintf("%d", time.Now().Unix()),
				}
				CurrRedisData.Records[addr] = tmpCur
			} else {
				tmpCur.EndTime = fmt.Sprintf("%d", time.Now().Unix())
			}
			continue
		}
		// 查看先前是否有记录
		tmpPre := PreRedisData.Records[addr]
		// 先前没记录
		if tmpPre == nil {
			tmpCur := CurrRedisData.Records[addr]
			if tmpCur == nil {
				// 且当前也没记录，添加新记录，但计数忽略
				tmpCur = &RedisDataRecord{
					StartTime: fmt.Sprintf("%d", time.Now().Unix()),
					EndTime:   fmt.Sprintf("%d", time.Now().Unix()),
				}
				CurrRedisData.Records[addr] = tmpCur
			} else {
				// 当前有记录，更新结束时间,忽略计数
				tmpCur.EndTime = fmt.Sprintf("%d", time.Now().Unix())
			}
		} else {
			// 先前有记录
			tmpCur := CurrRedisData.Records[addr]
			if tmpCur == nil {
				// 当前没记录,添加新记录并加差值
				tmpCur = &RedisDataRecord{
					StartTime: fmt.Sprintf("%d", time.Now().Unix()),
					EndTime:   fmt.Sprintf("%d", time.Now().Unix()),
				}
				CurrRedisData.Records[addr] = tmpCur
			}
			tmpCur.EndTime = fmt.Sprintf("%d", time.Now().Unix())
			if record.OpNum >= tmpPre.OpNum {
				tmpCur.OpNum += (record.OpNum - tmpPre.OpNum)
			} else {
				tmpCur.OpNum += record.OpNum
			}
			if record.OpFailNum >= tmpPre.OpFailNum {
				tmpCur.OpFailNum += (record.OpFailNum - tmpPre.OpFailNum)
			} else {
				tmpCur.OpFailNum += record.OpFailNum
			}
		}
	}

	// 根据临时数据差值更新当前统计(cmd数据)
	for cmdname, cmdinfo := range TmpRedisCmd.Cmds {
		if PreRedisCmd == nil {
			continue
		}
		if PreRedisCmd.ProxyAddr != TmpRedisCmd.ProxyAddr {
			continue
		}
		// 查看先前是否有记录
		tmpPre := PreRedisCmd.Cmds[cmdname]
		// 先前没记录
		if tmpPre == nil {
			continue
		} else {
			// 先前有记录
			tmpCur := CurrRedisCmd.Cmds[cmdname]
			if tmpCur == nil {
				// 当前没记录,添加新记录并加差值
				tmpCur = &RedisCmdRecord{
					StartTime: fmt.Sprintf("%d", time.Now().Unix()),
					EndTime:   fmt.Sprintf("%d", time.Now().Unix()),
				}
				CurrRedisCmd.Cmds[cmdname] = tmpCur
			}
			tmpCur.EndTime = fmt.Sprintf("%d", time.Now().Unix())
			if cmdinfo.OpNum >= tmpPre.OpNum {
				tmpCur.OpNum += (cmdinfo.OpNum - tmpPre.OpNum)
			} else {
				tmpCur.OpNum += cmdinfo.OpNum
			}
			if cmdinfo.OpSecs >= tmpPre.OpSecs {
				tmpCur.OpSecs += (cmdinfo.OpSecs - tmpPre.OpSecs)
			} else {
				tmpCur.OpSecs += cmdinfo.OpSecs
			}
			if cmdinfo.OpFailNum >= tmpPre.OpFailNum {
				tmpCur.OpFailNum += (cmdinfo.OpFailNum - tmpPre.OpFailNum)
			} else {
				tmpCur.OpFailNum += cmdinfo.OpFailNum
			}
			if cmdinfo.OpFailSecs >= tmpPre.OpFailSecs {
				tmpCur.OpFailSecs += (cmdinfo.OpFailSecs - tmpPre.OpFailSecs)
			} else {
				tmpCur.OpFailSecs += cmdinfo.OpFailSecs
			}
		}
	}

	// 记录上次统计值
	StatisPreRedisData[index] = TmpRedisData
	StatisPreRedisCmd[index] = TmpRedisCmd

	// 打印当前数据
	//GLogger.Printf("----Print cur redis data---")
	//PrintRedisData(&CurrRedisData)
	//GLogger.Printf("----Print cur redis cmd----")
	//PrintRedisCmd(&CurrRedisCmd)
}

func GenStatisRedisDataName() string {
	return fmt.Sprintf(".statistics_redis_data.%s", PreTimeFlag)
}

func GenStatisRedisCmdName() string {
	return fmt.Sprintf(".statistics_redis_cmd.%s", PreTimeFlag)
}

func GenNowStatisRedisDataName() string {
	return fmt.Sprintf(".statistics_redis_data.%s", TodayStr)
}

func GenNowStatisRedisCmdName() string {
	return fmt.Sprintf(".statistics_redis_cmd.%s", TodayStr)
}

func SaveCurRedisRecord() {
	// redis data statistics
	tmpfile := GenNowStatisRedisDataName()
	SaveCurRedisData(tmpfile, &CurrRedisData)

	// redis cmd statistics
	tmpfile2 := GenNowStatisRedisCmdName()
	SaveCurRedisCmd(tmpfile2, &CurrRedisCmd)
}

func SavePreRedisRecord() {
	tmpfile1 := GenNowPreRedisDataName()
	SaveRedisStatisticData(tmpfile1, StatisPreRedisData)

	tmpfile2 := GenNowPreRedisCmdName()
	SaveRedisStatisticCmd(tmpfile2, StatisPreRedisCmd)
}

func LoadRedisData(filename string) *RedisDataStatistic {
	var tmpData *RedisDataStatistic = &RedisDataStatistic{
		Records: make(map[string]*RedisDataRecord),
	}
	tmpfile1, e1 := os.Open(filename)
	if e1 != nil {
		GLogger.Printf("GetRedisData [%s] failed,err:%s", filename, e1.Error())
		return nil
	}
	defer tmpfile1.Close()
	rd := bufio.NewReader(tmpfile1)
	for {
		lineData, err := rd.ReadString('\n')
		if err != nil || err == io.EOF {
			break
		}
		var redisaddr string
		var tmpRedisData *RedisDataRecord = &RedisDataRecord{}
		res, e := fmt.Sscanf(lineData, "%s\t%s\t%s\t%d\t%d\n", &redisaddr, &tmpRedisData.StartTime, &tmpRedisData.EndTime,
			&tmpRedisData.OpNum, &tmpRedisData.OpFailNum)
		if res != 5 && e != nil {
			GLogger.Printf("LoadCurrRedisInfo sscanf [%s] failed,err:%s", lineData, e.Error())
			continue
		}
		tmp := tmpData.Records[redisaddr]
		if tmp == nil {
			tmpData.Records[redisaddr] = tmpRedisData
		}
	}
	return tmpData
}

func LoadRedisCmd(filename string) *RedisCmdStatistic {
	var tmpCmd *RedisCmdStatistic = &RedisCmdStatistic{
		Cmds: make(map[string]*RedisCmdRecord),
	}
	// load redis cmd
	tmpfile2, e2 := os.Open(filename)
	if e2 != nil {
		GLogger.Printf("GetRedisCmd [%s] failed,err:%s", filename, e2.Error())
		return nil
	}
	defer tmpfile2.Close()
	rd2 := bufio.NewReader(tmpfile2)
	for {
		lineData, err := rd2.ReadString('\n')
		if err != nil || err == io.EOF {
			break
		}
		var cmdname string
		var tmpRedisCmd *RedisCmdRecord = &RedisCmdRecord{}
		res, e := fmt.Sscanf(lineData, "%s\t%s\t%s\t%d\t%d\t%d\t%d\n", &cmdname, &tmpRedisCmd.StartTime, &tmpRedisCmd.EndTime,
			&tmpRedisCmd.OpNum, &tmpRedisCmd.OpSecs, &tmpRedisCmd.OpFailNum, &tmpRedisCmd.OpFailSecs)
		if res != 7 && e != nil {
			GLogger.Printf("LoadCurrRedisInfo sscanf [%s] failed,err:%s", lineData, e.Error())
			continue
		}
		tmp := tmpCmd.Cmds[cmdname]
		if tmp == nil {
			tmpCmd.Cmds[cmdname] = tmpRedisCmd
		}
	}
	return tmpCmd
}

// 启动系统后从文件中读取上次的redis统计记录
func LoadCurrRedisInfo() {
	// load redis data
	tmpfilename1 := GenNowStatisRedisDataName()
	tmpfile1, e1 := os.Open(tmpfilename1)
	if e1 != nil {
		GLogger.Printf("LoadCurrRedisInfo [%s] failed,err:%s", tmpfilename1, e1.Error())
		return
	}
	defer tmpfile1.Close()
	rd := bufio.NewReader(tmpfile1)
	for {
		lineData, err := rd.ReadString('\n')
		if err != nil || err == io.EOF {
			break
		}
		var redisaddr string
		var tmpRedisData *RedisDataRecord = &RedisDataRecord{}
		res, e := fmt.Sscanf(lineData, "%s\t%s\t%s\t%d\t%d\n", &redisaddr, &tmpRedisData.StartTime, &tmpRedisData.EndTime,
			&tmpRedisData.OpNum, &tmpRedisData.OpFailNum)
		if res != 5 && e != nil {
			GLogger.Printf("LoadCurrRedisInfo sscanf [%s] failed,err:%s", lineData, e.Error())
			continue
		}
		tmp := CurrRedisData.Records[redisaddr]
		if tmp == nil {
			CurrRedisData.Records[redisaddr] = tmpRedisData
		}
	}
	// load redis cmd
	tmpfilename2 := GenNowStatisRedisCmdName()
	tmpfile2, e2 := os.Open(tmpfilename2)
	if e2 != nil {
		GLogger.Printf("LoadCurrRedisInfo [%s] failed,err:%s", tmpfilename2, e2.Error())
		return
	}
	defer tmpfile2.Close()
	rd2 := bufio.NewReader(tmpfile2)
	for {
		lineData, err := rd2.ReadString('\n')
		if err != nil || err == io.EOF {
			break
		}
		var cmdname string
		var tmpRedisCmd *RedisCmdRecord = &RedisCmdRecord{}
		res, e := fmt.Sscanf(lineData, "%s\t%s\t%s\t%d\t%d\t%d\t%d\n", &cmdname, &tmpRedisCmd.StartTime, &tmpRedisCmd.EndTime,
			&tmpRedisCmd.OpNum, &tmpRedisCmd.OpSecs, &tmpRedisCmd.OpFailNum, &tmpRedisCmd.OpFailSecs)
		if res != 7 && e != nil {
			GLogger.Printf("LoadCurrRedisInfo sscanf [%s] failed,err:%s", lineData, e.Error())
			continue
		}
		tmp := CurrRedisCmd.Cmds[cmdname]
		if tmp == nil {
			CurrRedisCmd.Cmds[cmdname] = tmpRedisCmd
		}
	}
}

/*
func LoadPreRedisInfo() {
	// load redis data
	tmpfilename1 := GenPreRedisDataName()
	tmpfile1, e1 := os.Open(tmpfilename1)
	if e1 != nil {
		GLogger.Printf("LoadPreRedisInfo [%s] failed,err:%s", tmpfilename1, e1.Error())
		return
	}
	defer tmpfile1.Close()
	rd := bufio.NewReader(tmpfile1)
	for {
		lineData, err := rd.ReadString('\n')
		if err != nil || err == io.EOF {
			break
		}
		var redisaddr string
		var tmpRedisData *RedisDataRecord = &RedisDataRecord{}
		res, e := fmt.Sscanf(lineData, "%s\t%s\t%s\t%d\t%d\n", &redisaddr, &tmpRedisData.StartTime, &tmpRedisData.EndTime,
			&tmpRedisData.OpNum, &tmpRedisData.OpFailNum)
		if res != 5 && e != nil {
			GLogger.Printf("LoadPreRedisInfo sscanf [%s] failed,err:%s", lineData, e.Error())
			continue
		}
		if PreRedisData == nil {
			PreRedisData = &RedisDataStatistic{
				Records: make(map[string]*RedisDataRecord),
			}
		}
		tmp := PreRedisData.Records[redisaddr]
		if tmp == nil {
			PreRedisData.Records[redisaddr] = tmpRedisData
		}
	}
	// load redis cmd
	tmpfilename2 := GenPreRedisCmdName()
	tmpfile2, e2 := os.Open(tmpfilename2)
	if e2 != nil {
		GLogger.Printf("LoadPreRedisInfo [%s] failed,err:%s", tmpfilename2, e2.Error())
		return
	}
	defer tmpfile2.Close()
	rd2 := bufio.NewReader(tmpfile2)
	for {
		lineData, err := rd2.ReadString('\n')
		if err != nil || err == io.EOF {
			break
		}
		var cmdname string
		var tmpRedisCmd *RedisCmdRecord = &RedisCmdRecord{}
		res, e := fmt.Sscanf(lineData, "%s\t%s\t%s\t%d\t%d\t%d\t%d\n", &cmdname, &tmpRedisCmd.StartTime, &tmpRedisCmd.EndTime,
			&tmpRedisCmd.OpNum, &tmpRedisCmd.OpSecs, &tmpRedisCmd.OpFailNum, &tmpRedisCmd.OpFailSecs)
		if res != 7 && e != nil {
			GLogger.Printf("LoadPreRedisInfo sscanf [%s] failed,err:%s", lineData, e.Error())
			continue
		}
		if PreRedisCmd == nil {
			PreRedisCmd = &RedisCmdStatistic{
				Cmds: make(map[string]*RedisCmdRecord),
			}
		}
		tmp := PreRedisCmd.Cmds[cmdname]
		if tmp == nil {
			PreRedisCmd.Cmds[cmdname] = tmpRedisCmd
		}
	}
}
*/
func ResetRedisInfo() {
	for addr, _ := range CurrRedisData.Records {
		delete(CurrRedisData.Records, addr)
	}
	for cmd, _ := range CurrRedisCmd.Cmds {
		delete(CurrRedisCmd.Cmds, cmd)
	}
}

func GenStatisProxyDataName() string {
	return fmt.Sprintf(".statistics_proxy_data.%s", PreTimeFlag)
}

func GenPreProxyDataName() string {
	return fmt.Sprintf(".pre_proxy_data.%s", PreTimeFlag)
}

func GenNowStatisProxyDataName() string {
	return fmt.Sprintf(".statistics_proxy_data.%s", TodayStr)
}

func GenNowPreProxyDataName() string {
	return fmt.Sprintf(".pre_proxy_data.%s", TodayStr)
}

func SavePreProxyRecord() {
	nowproxyfile := GenNowPreProxyDataName()
	tmpfile, err := os.OpenFile(nowproxyfile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if tmpfile == nil {
		GLogger.Printf("open file [%s] failed,err:%s", nowproxyfile, err.Error())
	} else {
		defer tmpfile.Close()
		var i int = 0
		for _, proxy := range PreStatisticProxyData {
			if proxy == nil {
				continue
			}
			//GLogger.Printf("start[%s],end[%s]", CurrProxyData[i].StartTime, CurrProxyData[i].EndTime)
			s := fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\n", proxy.Addr, proxy.StartTime,
				proxy.EndTime, proxy.ConnNum, proxy.ConnFailNum,
				proxy.OpNum, proxy.OpFailNum, proxy.OpSuccNum)
			tmpfile.WriteString(s)
			//GLogger.Printf("cur proxy data %s [%s]", time.Now().String(), s)
			i++
		}
	}
}

func SaveCurProxyRecord() {
	nowproxyfile := GenNowStatisProxyDataName()
	tmpfile, err := os.OpenFile(nowproxyfile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if tmpfile == nil {
		GLogger.Printf("open file [%s] failed,err:%s", nowproxyfile, err.Error())
	} else {
		defer tmpfile.Close()
		var i int = 0
		for _, proxy := range CurrProxyData {
			if proxy == nil {
				continue
			}
			//GLogger.Printf("start[%s],end[%s]", CurrProxyData[i].StartTime, CurrProxyData[i].EndTime)
			s := fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\n", proxy.Addr, proxy.StartTime,
				proxy.EndTime, proxy.ConnNum, proxy.ConnFailNum,
				proxy.OpNum, proxy.OpFailNum, proxy.OpSuccNum)
			tmpfile.WriteString(s)
			//GLogger.Printf("cur proxy data %s [%s]", time.Now().String(), s)
			i++
		}
	}
}

func PrintCurrProxyData(proxydata [64]*ProxyPerDataNode) {
	//GLogger.Print("--------Now print proxy data.------")
	for index, proxy := range proxydata {
		if proxy != nil {
			GLogger.Printf("proxy[%d],addr[%s],start[%s],end[%s],connnum[%d],connfailnum[%d],opnum[%d],opfailnum[%d],opsucnum[%d]",
				index, proxy.Addr, proxy.StartTime, proxy.EndTime, proxy.ConnNum, proxy.ConnFailNum, proxy.OpNum,
				proxy.OpFailNum, proxy.OpSuccNum)
		}
	}
	//GLogger.Print("-------------------------------------------")
}

func LoadPreProxyData() {
	tmpfilename := GenPreProxyDataName()
	tmpfile, e := os.Open(tmpfilename)
	if e != nil {
		GLogger.Printf("LoadPreProxyData [%s] failed,err:%s.", tmpfilename, e.Error())
		return
	}
	defer tmpfile.Close()
	rd := bufio.NewReader(tmpfile)
	for {
		lineData, err := rd.ReadString('\n')
		if err != nil || err == io.EOF {
			break
		}
		//GLogger.Printf("now will sscanf proxy data")
		var proxyAddr string
		var tmpProxyData *ProxyPerDataNode = &ProxyPerDataNode{}
		res, e := fmt.Sscanf(lineData, "%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\n", &proxyAddr, &tmpProxyData.StartTime, &tmpProxyData.EndTime,
			&tmpProxyData.ConnNum, &tmpProxyData.ConnFailNum, &tmpProxyData.OpNum, &tmpProxyData.OpFailNum,
			&tmpProxyData.OpSuccNum)
		if res != 8 && e != nil {
			GLogger.Printf("LoadPreProxyData sscanf [%s] failed,err:%s.", lineData, e.Error())
			continue
		}
		tmpProxyData.Addr = proxyAddr
		tmpProxyData.StartTime = fmt.Sprintf("%d", time.Now().Unix())
		tmpProxyData.EndTime = fmt.Sprintf("%d", time.Now().Unix())
		//GLogger.Printf("now will find in proxylist")
		var i int = 0
		for _, proxy := range GlobalConfig.ProxyList {
			//GLogger.Printf("compare [%s] and [%s]", proxyAddr, proxy.ProxyAddr)
			if strings.Compare(proxyAddr, proxy.ProxyAddr) == 0 {
				tmp := PreStatisticProxyData[i]
				if tmp == nil {
					PreStatisticProxyData[i] = tmpProxyData
				}
				//GLogger.Printf("Load pre statistics for [%s] success.", proxyAddr)
			}
			i++
		}
	}
	//GLogger.Printf("Load Pre proxy data success.")
	//PrintCurrProxyData(PreStatisticProxyData)
}

// 系统启动时从文件中读取上次的proxy统计记录
func LoadCurrProxyData() {
	tmpfilename := GenNowStatisProxyDataName()
	tmpfile, e := os.Open(tmpfilename)
	if e != nil {
		GLogger.Printf("LoadCurrProxyData [%s] failed,err:%s.", tmpfilename, e.Error())
		return
	}
	defer tmpfile.Close()
	rd := bufio.NewReader(tmpfile)
	for {
		lineData, err := rd.ReadString('\n')
		if err != nil || err == io.EOF {
			break
		}
		//GLogger.Printf("now will sscanf proxy data")
		var proxyAddr string
		var tmpProxyData *ProxyPerDataNode = &ProxyPerDataNode{}
		res, e := fmt.Sscanf(lineData, "%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\n", &proxyAddr, &tmpProxyData.StartTime, &tmpProxyData.EndTime,
			&tmpProxyData.ConnNum, &tmpProxyData.ConnFailNum, &tmpProxyData.OpNum, &tmpProxyData.OpFailNum,
			&tmpProxyData.OpSuccNum)
		if res != 8 && e != nil {
			GLogger.Printf("LoadCurrProxyData sscanf [%s] failed,err:%s.", lineData, e.Error())
			continue
		}
		tmpProxyData.Addr = proxyAddr
		tmpProxyData.StartTime = fmt.Sprintf("%d", time.Now().Unix())
		tmpProxyData.EndTime = fmt.Sprintf("%d", time.Now().Unix())
		//GLogger.Printf("now will find in proxylist")
		var i int = 0
		for _, proxy := range GlobalConfig.ProxyList {
			//GLogger.Printf("compare [%s] and [%s]", proxyAddr, proxy.ProxyAddr)
			if strings.Compare(proxyAddr, proxy.ProxyAddr) == 0 {
				tmp := CurrProxyData[i]
				if tmp == nil {
					CurrProxyData[i] = tmpProxyData
				}
				//GLogger.Printf("Load pre statistics for [%s] success.", proxyAddr)
			}
			i++
		}
	}
	//GLogger.Printf("Load Curr proxy data success.")
	//PrintCurrProxyData(CurrProxyData)
}

func ResetProxyData() {
	var i int = 0
	for _, _ = range GlobalConfig.ProxyList {
		tmp := CurrProxyData[i]
		if tmp != nil {
			tmp.ConnNum = 0
			tmp.ConnFailNum = 0
			tmp.OpNum = 0
			tmp.OpFailNum = 0
			tmp.OpSuccNum = 0
			tmp.StartTime = fmt.Sprintf("%d", time.Now().Unix())
		}
		i++
	}
}

func usage(progName string) {
	fmt.Printf("Usage:%s or %s configFile\n", progName, progName)
	os.Exit(0)
}

func GetTimeStr(t time.Time) string {
	return fmt.Sprintf("%04d%02d%02d", t.Year(), t.Month(), t.Day())
}

func SendDayReport2(conf *MoniDataConf) error {
	// get proxy day data statistics
	var proxyData string = ""
	var redisData string = ""
	var redisCmd string = ""
	proxyDataFileName := GenStatisProxyDataName()
	proxyDataNode := GetProxyDayData2(proxyDataFileName)

	proxyData = GenProxyDataHtml2(proxyDataNode)
	//GLogger.Printf("SendDayReport2 proxy data:%s", proxyData)

	// get redis day data statistics
	redisDataFileName := GenStatisRedisDataName()
	redisDataS := LoadRedisData(redisDataFileName)
	if redisDataS != nil {
		redisData = GenRedisData2(redisDataS)
	}
	//GLogger.Printf("SendDayReport2 redis data:%s", redisData)

	// get redis day cmd statistics
	redisCmdFileName := GenStatisRedisCmdName()
	redisCmdS := LoadRedisCmd(redisCmdFileName)
	if redisCmdS != nil {
		redisCmd = GenRedisCmd2(redisCmdS)
	}
	//GLogger.Printf("SendDayReport2 redis cmd:%s", redisCmd)
	html := GenDaySummaryReportHtml(proxyData, redisData, redisCmd)
	html = strings.Replace(html, " >", ">", -1)
	var Subject string = "Codis集群监控统计 (" + PreTimeFlag + ")"
	GLogger.Printf("email[%s],content\n%s\n", Subject, html)
	for _, addr := range strings.Split(GlobalConfig.ToAddr, ";") {
		err := SendSmtpEmail(GlobalConfig.EmailAddr, GlobalConfig.EmailPwd, GlobalConfig.SmtpAddr, addr, Subject, html, "html")
		if err != nil {
			GLogger.Printf("Send Report Email [%s] to [%s] failed,err:%s\n", Subject, addr, err.Error())
		} else {
			GLogger.Printf("Send Report Email [%s] to [%s] success\n", Subject, addr)
		}
	}

	return nil
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
		proxyDataFileName := ".proxy_" + addr + "." + PreTimeFlag + ".data"
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
		/*proxyCmdFileName := ".proxy_" + addr + "." + PreTimeFlag + ".cmd"
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
		redisDataFileName := ".redis_" + tmpaddr + "." + PreTimeFlag + ".data"
		redisDataMap := GetRedisDayData(redisDataFileName, conf.MoniInterval)
		if redisDataMap == nil {
			continue
		}

		redisData += GenRedisSummaryHtml(tmpaddr, redisDataMap)
		//PrintRedisDataMap(redisDataMap)

		// Get redis day cmd
		redisCmdFileName := ".redis_" + tmpaddr + "." + PreTimeFlag + ".cmd"
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
	var Subject string = "Codis集群监控统计 (" + PreTimeFlag + ")"
	GLogger.Printf("report content:\n%s\n", html)
	err := SendSmtpEmail(GlobalConfig.EmailAddr, conf.EmailPwd, conf.SmtpAddr, conf.ToAddr, Subject, html, "html")
	if err != nil {
		GLogger.Printf("Send Email [%s] to [%s] failed,err:%s\n", Subject, GlobalConfig.ToAddr, err.Error())
	} else {
		GLogger.Printf("Send Email [%s] to [%s] success\n", Subject, GlobalConfig.ToAddr)
	}
	return nil
}

var PreTimeFlag string = ""

func CheckDate() {
	var NowTime time.Time = time.Now()
	var tmpStr string = GetTimeStr(NowTime)

	if TodayStr != tmpStr {
		PreTimeFlag = TodayStr
		write_date(PreTimeFlag)
		go func() {
			for {
				var t time.Time = time.Now()
				if strconv.FormatInt(int64(t.Hour()), 10) != GlobalConfig.SendTime {
					time.Sleep(time.Second * 60)
					continue
				}
				break
			}
			err := SendDayReport2(GlobalConfig)
			if err != nil {
				GLogger.Printf("send report failed,err:%s\n", err.Error())
			}
		}()
		TodayStr = tmpStr

		// reset proxy info
		ResetProxyData()
		// reset redis info
		ResetRedisInfo()
	}
}

var LogFileName = "MoniCoids.log"
var GLogger *log.Logger
var ConfFile string = "monidata.json"
var GlobalConfig *MoniDataConf = NewConf()
var TodayTime time.Time = time.Now()
var TodayStr string = GetTimeStr(TodayTime)

func TestSendReport() {
	PreTimeFlag = TodayStr
	/*for i := 0; i < 300; i++ {
		time.Sleep(time.Second)
	}*/

	SendDayReport2(GlobalConfig)

	os.Exit(0)
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
		if strings.Contains(path, ".proxy_") || strings.Contains(path, ".redis_") ||
			strings.Contains(path, ".statistics_") || strings.Contains(path, ".pre_") {
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
	PreTimeFlag = read_date()
	GLogger.Printf("get pre date=%s", PreTimeFlag)
	// Load config
	err := LoadConf(ConfFile, &GlobalConfig)
	if err != nil {
		os.Exit(2)
	}
	GLogger.Printf("[%s] start.", os.Args[0])
	// Load pre proxy statistics
	LoadCurrProxyData()
	LoadPreProxyData()

	//LoadCurrRedisInfo()
	//LoadPreRedisInfo()
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
