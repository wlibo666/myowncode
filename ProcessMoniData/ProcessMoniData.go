package main

import (
	"encoding/json"
	//	"errors"
	"fmt"
	io "io/ioutil"
	"os"
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
	fmt.Printf("json data\n %s \n", string(data))
	err = json.Unmarshal([]byte(data), v)
	if err != nil {
		fmt.Printf("Unmarshal json failed,errr:%s\n", err.Error())
		return err
	}
	fmt.Printf("struct config:%v\n", v)
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
	err := LoadConf(ConfFile, &GlobalConfig)
	if err != nil {
		os.Exit(2)
	}
	PrintConf(GlobalConfig)

}
