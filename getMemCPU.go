package main

import (
	"fmt"
	"github.com/shirou/gopsutil/mem"
)

func main() {
	v, _ := mem.VirtualMemory()

	fmt.Printf("Total:%v,free:%v,usedpercent:%v\n", v.Total, v.Free, v.UsedPercent)
	fmt.Printf("%v\n", v)
}
