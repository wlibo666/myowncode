package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
)

var dashboardaddr string = ""

const (
	METHOD_GET    HttpMethod = "GET"
	METHOD_POST   HttpMethod = "POST"
	METHOD_PUT    HttpMethod = "PUT"
	METHOD_DELETE HttpMethod = "DELETE"
)

type HttpMethod string

func jsonify(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

func callApi(method HttpMethod, apiPath string, params interface{}, retVal interface{}) error {
	if apiPath[0] != '/' {
		fmt.Printf("api path must starts with /\n")
		os.Exit(1)
	}
	url := "http://" + dashboardaddr + apiPath
	client := &http.Client{Transport: http.DefaultTransport}

	b, err := json.Marshal(params)
	if err != nil {
		fmt.Printf("Marshal [%v] failed,err:%s\n", params, err.Error())
		os.Exit(1)
	}

	req, err := http.NewRequest(string(method), url, strings.NewReader(string(b)))
	if err != nil {
		fmt.Printf("NewRequest failed,err:%s\n", err.Error())
		os.Exit(1)
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("can't connect to dashboard, please check 'dashboard_addr[%s]' is corrent in config file\n", dashboardaddr)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("ReadAll failed,err:%s\n", err.Error())
		os.Exit(1)
	}

	if resp.StatusCode == 200 {
		err := json.Unmarshal(body, retVal)
		if err != nil {
			fmt.Printf("Unmarshal [%v] failed,err:%s\n", body, err.Error())
			os.Exit(1)
		}
		return nil
	}
	fmt.Printf("http status code %d, %s\n", resp.StatusCode, string(body))
	os.Exit(1)
	return nil
}

type RangeSetTask struct {
	FromSlot   int    `json:"from"`
	ToSlot     int    `json:"to"`
	NewGroupId int    `json:"new_group"`
	Status     string `json:"status"`
}

func runSlotRangeSet(fromSlotId, toSlotId int, groupId int, status string) error {
	t := RangeSetTask{
		FromSlot:   fromSlotId,
		ToSlot:     toSlotId,
		NewGroupId: groupId,
		Status:     status,
	}
	var v interface{}
	err := callApi(METHOD_POST, "/api/slot", t, &v)
	if err != nil {
		return nil
	}
	fmt.Printf("set slot[%d - %d] to group [%d] ,status [%s] success.\n", fromSlotId, toSlotId, groupId, status)
	return nil
}

type SlotStatus string

const (
	SLOT_STATUS_ONLINE      SlotStatus = "online"
	SLOT_STATUS_OFFLINE     SlotStatus = "offline"
	SLOT_STATUS_MIGRATE     SlotStatus = "migrate"
	SLOT_STATUS_PRE_MIGRATE SlotStatus = "pre_migrate"
)

func SetSlotGroup(slotFrom int, slotTo int, groupid int) {
	runSlotRangeSet(slotFrom, slotTo, groupid, string(SLOT_STATUS_ONLINE))
}

var oldgroup int
var newgroup int

func ParseFileAndSet(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("open file [%s] failed,err:%s\n", filename, err.Error())
		os.Exit(1)
	}
	defer file.Close()
	rd := bufio.NewReader(file)
	for {
		lineData, err := rd.ReadString('\n')
		if err != nil || err == io.EOF {
			break
		}
		var slotid int
		var groupid int
		var res int
		res, err = fmt.Sscanf(lineData, "%d\t%d\n", &slotid, &groupid)
		if err != nil && res != 2 {
			fmt.Printf("err:%s,config line [%s] is incorrect\n", err.Error(), lineData)
			os.Exit(1)
		}
		if groupid == oldgroup {
			SetSlotGroup(slotid, slotid, newgroup)
		}
	}

}

func Usage(cmd string) {
	fmt.Printf("Usage: %s slotgroupfile dashboadaddr oldgroupid newgroupid\n", cmd)
	fmt.Printf("    eg: %s 28.5.slot.group 10.98.28.5:18087 7 1\n", cmd)
	os.Exit(0)
}

func main() {
	if len(os.Args) != 5 {
		Usage(os.Args[0])
	}
	var err error
	oldgroup, err = strconv.Atoi(os.Args[3])
	if err != nil {
		fmt.Printf("oldgroup [%s] should be interger\n", os.Args[3])
		Usage(os.Args[0])
	}
	newgroup, err = strconv.Atoi(os.Args[4])
	if err != nil {
		fmt.Printf("newgroup [%s] should be interger\n", os.Args[4])
		Usage(os.Args[0])
	}
	dashboardaddr = os.Args[2]
	slotgroupfile := os.Args[1]
	ParseFileAndSet(slotgroupfile)
}
