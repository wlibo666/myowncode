package main

import (
	"encoding/json"
	//"errors"
	"fmt"
	io "io/ioutil"
	"net/http"
	"os"
	//"strings"
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
			}
			PrintMoniData(proxy.ProxyAddr, monidata)
		}
		for i := 0; i < conf.MoniInterval; i++ {
			time.Sleep(time.Second)
		}
		break
	}
	return nil
}

func usage(progName string) {
	fmt.Printf("Usage:%s or %s configFile\n", progName, progName)
	os.Exit(0)
}

var ConfFile string = "monidata.json"
var GlobalConfig *MoniDataConf = NewConf()

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

	for {
		time.Sleep(time.Second)
	}
}
