package main

import (
	md "../../codis/pkg/models"
	"fmt"
	"github.com/wandoulabs/zkhelper"
	"os"
	"strings"
)

type CodisConf struct {
	zkConn      zkhelper.Conn
	zkAddr      string
	productName string
}

var Gconf CodisConf = CodisConf{
	zkAddr:      "10.98.28.5:2181",
	productName: "pusher",
}

func Usage(cmd string) {
	fmt.Printf("Usage: %s zkaddr dbname\n", cmd)
	fmt.Printf("    eg: %s 10.98.28.5:2181 pusher\n", cmd)
	os.Exit(0)
}

func GetSlotInfo(zkConn zkhelper.Conn, productName string, slotid int) (*md.Slot, error) {
	s, err := md.GetSlot(zkConn, productName, slotid)
	if err != nil {
		fmt.Printf("GetSlot [%d] failed,err:%s\n", slotid, err.Error())
		os.Exit(1)
	}
	return s, nil
}

var MaxSlotNum int = 1024

func GetSlotGroup() {
	filename := fmt.Sprintf("%s.slot.group", strings.Split(Gconf.zkAddr, ":")[0])
	fmt.Printf("will store slot group info into file [%s]\n", filename)

	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		fmt.Printf("Open file [%s] failed,err:%s\n", filename, err.Error())
		os.Exit(1)
	}
	defer file.Close()

	Gconf.zkConn, err = zkhelper.ConnectToZk(Gconf.zkAddr, 30000)
	if err != nil {
		fmt.Printf("connect to zk [%s] failed,err:%s\n", Gconf.zkAddr, err.Error())
		os.Exit(1)
	}
	var i int = 0
	for i = 0; i < MaxSlotNum; i++ {
		sinfo, _ := GetSlotInfo(Gconf.zkConn, Gconf.productName, i)
		str := fmt.Sprintf("%d\t%d\n", i, sinfo.GroupId)
		file.WriteString(str)
	}
	Gconf.zkConn.Close()
}

// ./GetSlotGroup zkaddr dbname
func main() {
	if len(os.Args) != 3 {
		Usage(os.Args[0])
	}
	Gconf.zkAddr = os.Args[1]
	Gconf.productName = os.Args[2]

	GetSlotGroup()
}
