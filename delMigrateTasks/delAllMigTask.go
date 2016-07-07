package main

import (
	"fmt"
	"os"
)

var zkAddr string
var dbName string

func usage(program string) {
	fmt.Printf("%s zkAddr dbName\n", program)
}

func main() {
	if len(os.Args) != 3 {
		usage(os.Args[0])
		os.Exit(0)
	}
	zkAddr = os.Args[1]
	dbName = os.Args[2]

	err := DelAllTasks(zkAddr, dbName)
	if err != nil {
		fmt.Printf("delete task from zk[%s],db[%s] failed,err:%s\n",
			zkAddr, dbName, err.Error())
	}
}
