package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

func GenHtmlPage() string {
	return ""
}

type ProxyPerDataNode struct {
	TimeInterval int
	StartTime    string
	EndTime      string
	ConnNum      int64
	ConnFailNum  int64
	OpNum        int64
	OpFailNum    int64
	OpSuccNum    int64
}

type ProxyPerCmdNode struct {
	StartTime    string
	Cmd          string
	Calls        int
	FailCalls    int
	FailUsecs    int
	Usecs        int
	UsecsPerCall int
}

type CmdMap struct {
	TimeInterval int
	StartTime    string
	EndTime      string
	Cmds         map[string]*ProxyPerCmdNode
}

func GetProxyDayData(proxyDataFile string, interval int) *ProxyPerDataNode {
	file, err := os.OpenFile(proxyDataFile, os.O_RDONLY, os.ModePerm)
	if err != nil {
		fmt.Printf("open file [%s] failed,err:%s\n", proxyDataFile, err.Error())
		return nil
	}
	defer file.Close()
	var firstLine string
	var lastLine string
	var firstNode ProxyPerDataNode
	var lastNode ProxyPerDataNode
	var returnNode *ProxyPerDataNode
	var flag bool = false
	rd := bufio.NewReader(file)
	for {
		lineData, err := rd.ReadString('\n')
		if err != nil || err == io.EOF {
			break
		}
		if flag == false {
			firstLine = lineData
			lastLine = lineData
			flag = true
		} else {
			lastLine = lineData
		}
	}
	//fmt.Printf("firstLine:%s", firstLine)
	n1, err1 := fmt.Sscanf(firstLine, "%s\t%d\t%d\t%d\t%d\t%d\n",
		&firstNode.StartTime, &firstNode.ConnNum, &firstNode.ConnFailNum,
		&firstNode.OpNum, &firstNode.OpFailNum, &firstNode.OpSuccNum)
	if n1 != 6 {
		fmt.Printf("format from :[%s] failed, err:%s\n", firstLine, err1.Error())
		return nil
	}
	/*fmt.Printf("time:%s,ConnNum:%d,ConnFailNum:%d,OpNum:%d,OpFailNum:%d,OpSuccNum:%d\n",
		firstNode.StartTime, firstNode.ConnNum, firstNode.ConnFailNum,
		firstNode.OpNum, firstNode.OpFailNum, firstNode.OpSuccNum)
	fmt.Printf("lastLine:%s", lastLine)*/
	n2, err2 := fmt.Sscanf(lastLine, "%s\t%d\t%d\t%d\t%d\t%d\n",
		&lastNode.StartTime, &lastNode.ConnNum, &lastNode.ConnFailNum,
		&lastNode.OpNum, &lastNode.OpFailNum, &lastNode.OpSuccNum)
	if n2 != 6 {
		fmt.Printf("format from :[%s] failed, err:%s\n", lastLine, err2.Error())
		return nil
	}
	returnNode = &ProxyPerDataNode{
		TimeInterval: interval,
		StartTime:    firstNode.StartTime,
		EndTime:      lastNode.EndTime,
		ConnNum:      lastNode.ConnNum - firstNode.ConnNum,
		ConnFailNum:  lastNode.ConnFailNum - firstNode.ConnFailNum,
		OpNum:        lastNode.OpNum - firstNode.OpNum,
		OpFailNum:    lastNode.OpFailNum - firstNode.OpFailNum,
		OpSuccNum:    lastNode.OpSuccNum - firstNode.OpSuccNum,
	}
	return returnNode
}

func PrintCmdMap(cmds *CmdMap) {
	fmt.Printf("--------------------------------\n")
	fmt.Printf("TimeInterval:%d\n", cmds.TimeInterval)
	fmt.Printf("StartTime:%s\n", cmds.StartTime)
	fmt.Printf("EndTime:%s\n", cmds.EndTime)
	for key, node := range cmds.Cmds {
		fmt.Printf("key:%s\n", key)
		fmt.Printf("StartTime:%s\n", node.StartTime)
		fmt.Printf("Cmd:%s\n", node.Cmd)
		fmt.Printf("Calls:%d\n", node.Calls)
		fmt.Printf("FailCalls:%d\n", node.FailCalls)
		fmt.Printf("FailUsecs:%d\n", node.FailUsecs)
		fmt.Printf("Usecs:%d\n", node.Usecs)
		fmt.Printf("UsecsPerCall:%d\n\n", node.UsecsPerCall)
	}
}

func GetProxyDayCmd(proxyCmdFile string, interval int) *CmdMap {
	file, err := os.OpenFile(proxyCmdFile, os.O_RDONLY, os.ModePerm)
	if err != nil {
		fmt.Printf("open file [%s] failed,err:%s\n", proxyCmdFile, err.Error())
		return nil
	}
	defer file.Close()
	var firstLine string
	var lastLine string
	var firstNode CmdMap = CmdMap{Cmds: make(map[string]*ProxyPerCmdNode)}
	var lastNode CmdMap = CmdMap{Cmds: make(map[string]*ProxyPerCmdNode)}
	var returnNode *CmdMap
	var flag bool = false
	var firstFlag bool = false
	var firstTimeStr string
	rd := bufio.NewReader(file)
	// Get first all cmds
	for {
		lineData, err := rd.ReadString('\n')
		if err != nil || err == io.EOF {
			break
		}
		if flag == false {
			firstLine = lineData
			// parse firstLine
			tmpNode := &ProxyPerCmdNode{}
			n1, err1 := fmt.Sscanf(firstLine, "%s\t%s\t%d\t%d\t%d\t%d\t%d\n",
				&tmpNode.StartTime, &tmpNode.Cmd, &tmpNode.Calls,
				&tmpNode.FailCalls, &tmpNode.FailUsecs,
				&tmpNode.Usecs, &tmpNode.UsecsPerCall)
			if n1 != 7 {
				fmt.Printf("sscanf form [%s] failed,err:%s\n", firstLine, err1.Error())
				continue
			}
			if firstFlag == false {
				firstTimeStr = tmpNode.StartTime
				firstFlag = true
			}
			if strings.Contains(lineData, firstTimeStr) == false {
				flag = true
			} else {
				//fmt.Printf("firstLine:%s\n", lineData)
				firstNode.Cmds[tmpNode.Cmd] = tmpNode
			}
		} else {
			lastLine = lineData
		}
	}
	//fmt.Printf("lastLine is:%s\n", lastLine)
	var lastTimeStr string
	n2, err2 := fmt.Sscanf(lastLine, "%s\t%*s\t%*s\t%*s\t%*s\t%*s\t%*s\n", &lastTimeStr)
	if n2 != 1 {
		fmt.Printf("Sscanf [%s] failed,err:%s\n", lastLine, err2.Error())
		return nil
	}
	//PrintCmdMap(&firstNode)
	// Get last all cmds
	file.Seek(0, os.SEEK_SET)
	//fmt.Printf("now read last node\n")
	rd = bufio.NewReader(file)
	for {
		lineData, err := rd.ReadString('\n')
		if err != nil || err == io.EOF {
			break
		}
		if strings.Contains(lineData, lastTimeStr) {
			tmpNode := &ProxyPerCmdNode{}
			n3, err3 := fmt.Sscanf(lineData, "%s\t%s\t%d\t%d\t%d\t%d\t%d\n",
				&tmpNode.StartTime, &tmpNode.Cmd, &tmpNode.Calls,
				&tmpNode.FailCalls, &tmpNode.FailUsecs,
				&tmpNode.Usecs, &tmpNode.UsecsPerCall)
			if n3 != 7 {
				fmt.Printf("sscanf form [%s] failed,err:%s\n", lineData, err3.Error())
				continue
			}
			lastNode.Cmds[tmpNode.Cmd] = tmpNode
		}

	}
	//PrintCmdMap(&lastNode)
	// calc oneday's cmds
	returnNode = &CmdMap{
		TimeInterval: interval,
		StartTime:    firstTimeStr,
		EndTime:      lastTimeStr,
		Cmds:         make(map[string]*ProxyPerCmdNode),
	}
	for key, node := range lastNode.Cmds {
		tmpNode := &ProxyPerCmdNode{
			Cmd: key,
		}
		tmpBaseNode := firstNode.Cmds[key]
		if tmpBaseNode != nil {
			tmpNode.Calls = node.Calls - tmpBaseNode.Calls
			tmpNode.FailCalls = node.FailCalls - tmpBaseNode.FailCalls
			tmpNode.Usecs = node.Usecs - tmpBaseNode.Usecs
			tmpNode.FailUsecs = node.FailUsecs - tmpBaseNode.FailUsecs
		} else {
			tmpNode.Calls = node.Calls
			tmpNode.FailCalls = node.FailCalls
			tmpNode.Usecs = node.Usecs
			tmpNode.FailUsecs = node.FailUsecs
		}
		returnNode.Cmds[key] = tmpNode
	}

	return returnNode
}
