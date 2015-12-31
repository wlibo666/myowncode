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
	Calls        int64
	FailCalls    int64
	FailUsecs    int64
	Usecs        int64
	UsecsPerCall int64
}

type ProxyCmdMap struct {
	TimeInterval int
	StartTime    string
	EndTime      string
	Calls        int64
	FailCalls    int64
	FailUsecs    int64
	Usecs        int64
	Cmds         map[string]*ProxyPerCmdNode
}

type RedisPerDataNode struct {
	StartTime string
	ProxyAddr string
	Calls     int64
	FailCalls int64
}

type RedisDataMap struct {
	TimeInterval int
	StartTime    string
	EndTime      string
	Calls        int64
	FailCalls    int64
	Datas        map[string]*RedisPerDataNode
}

type RedisPerCmdNode struct {
	StartTime    string
	ProxyAddr    string
	Cmd          string
	Calls        int64
	FailCalls    int64
	FailUsecs    int64
	Usecs        int64
	UsecsPerCall int64
}

type RedisCmdProxy struct {
	ProxyAddr string
	Calls     int64
	FailCalls int64
	FailUsecs int64
	Usecs     int64
	Cmds      map[string]*RedisPerCmdNode
}

type RedisCmdMap struct {
	TimeInterval int
	StartTime    string
	EndTime      string
	Calls        int64
	FailCalls    int64
	FailUsecs    int64
	Usecs        int64
	Proxys       map[string]*RedisCmdProxy
}

func GetProxyDayData(proxyDataFile string, interval int) *ProxyPerDataNode {
	file, err := os.OpenFile(proxyDataFile, os.O_RDONLY, os.ModePerm)
	if err != nil {
		GLogger.Printf("open file [%s] failed,err:%s\n", proxyDataFile, err.Error())
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
	//GLogger.Printf("firstLine:%s", firstLine)
	n1, err1 := fmt.Sscanf(firstLine, "%s\t%d\t%d\t%d\t%d\t%d\n",
		&firstNode.StartTime, &firstNode.ConnNum, &firstNode.ConnFailNum,
		&firstNode.OpNum, &firstNode.OpFailNum, &firstNode.OpSuccNum)
	if n1 != 6 {
		GLogger.Printf("format from :[%s] failed, err:%s\n", firstLine, err1.Error())
		return nil
	}
	/*GLogger.Printf("time:%s,ConnNum:%d,ConnFailNum:%d,OpNum:%d,OpFailNum:%d,OpSuccNum:%d\n",
		firstNode.StartTime, firstNode.ConnNum, firstNode.ConnFailNum,
		firstNode.OpNum, firstNode.OpFailNum, firstNode.OpSuccNum)
	GLogger.Printf("lastLine:%s", lastLine)*/
	n2, err2 := fmt.Sscanf(lastLine, "%s\t%d\t%d\t%d\t%d\t%d\n",
		&lastNode.StartTime, &lastNode.ConnNum, &lastNode.ConnFailNum,
		&lastNode.OpNum, &lastNode.OpFailNum, &lastNode.OpSuccNum)
	if n2 != 6 {
		GLogger.Printf("format from :[%s] failed, err:%s\n", lastLine, err2.Error())
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

func PrintProxyCmdMap(cmds *ProxyCmdMap) {
	GLogger.Printf("-----------------Proxy Cmd Map---------------\n")
	GLogger.Printf("TimeInterval:%d\n", cmds.TimeInterval)
	GLogger.Printf("StartTime:%s\n", cmds.StartTime)
	GLogger.Printf("EndTime:%s\n", cmds.EndTime)
	GLogger.Printf("all Calls:%d\n", cmds.Calls)
	GLogger.Printf("all FailCalls:%d\n", cmds.FailCalls)
	GLogger.Printf("all FailUsecs:%d\n", cmds.FailUsecs)
	GLogger.Printf("all Usecs:%d\n\n", cmds.Usecs)
	for key, node := range cmds.Cmds {
		GLogger.Printf("key:%s\n", key)
		GLogger.Printf("StartTime:%s\n", node.StartTime)
		GLogger.Printf("Cmd:%s\n", node.Cmd)
		GLogger.Printf("Calls:%d\n", node.Calls)
		GLogger.Printf("FailCalls:%d\n", node.FailCalls)
		GLogger.Printf("FailUsecs:%d\n", node.FailUsecs)
		GLogger.Printf("Usecs:%d\n", node.Usecs)
		GLogger.Printf("UsecsPerCall:%d\n\n", node.UsecsPerCall)
	}
}

func GetProxyDayCmd(proxyCmdFile string, interval int) *ProxyCmdMap {
	file, err := os.OpenFile(proxyCmdFile, os.O_RDONLY, os.ModePerm)
	if err != nil {
		GLogger.Printf("open file [%s] failed,err:%s\n", proxyCmdFile, err.Error())
		return nil
	}
	defer file.Close()
	var firstLine string
	var lastLine string
	var firstNode ProxyCmdMap = ProxyCmdMap{Cmds: make(map[string]*ProxyPerCmdNode)}
	var lastNode ProxyCmdMap = ProxyCmdMap{Cmds: make(map[string]*ProxyPerCmdNode)}
	var returnNode *ProxyCmdMap
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
				GLogger.Printf("sscanf form [%s] failed,err:%s\n", firstLine, err1.Error())
				continue
			}
			if firstFlag == false {
				firstTimeStr = tmpNode.StartTime
				firstFlag = true
			}
			if strings.Contains(lineData, firstTimeStr) == false {
				flag = true
			} else {
				//GLogger.Printf("firstLine:%s\n", lineData)
				firstNode.Cmds[tmpNode.Cmd] = tmpNode
			}
		} else {
			lastLine = lineData
		}
	}
	//GLogger.Printf("lastLine is:%s\n", lastLine)
	var lastTimeStr string
	n2, err2 := fmt.Sscanf(lastLine, "%s\t%*s\t%*s\t%*s\t%*s\t%*s\t%*s\n", &lastTimeStr)
	if n2 != 1 {
		GLogger.Printf("Sscanf [%s] failed,err:%s\n", lastLine, err2.Error())
		return nil
	}
	//PrintProxyCmdMap(&firstNode)
	// Get last all cmds
	file.Seek(0, os.SEEK_SET)
	//GLogger.Printf("now read last node\n")
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
				GLogger.Printf("sscanf form [%s] failed,err:%s\n", lineData, err3.Error())
				continue
			}
			lastNode.Cmds[tmpNode.Cmd] = tmpNode
		}

	}
	//PrintProxyCmdMap(&lastNode)
	// calc oneday's cmds
	returnNode = &ProxyCmdMap{
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
		returnNode.Calls += tmpNode.Calls
		returnNode.FailCalls += tmpNode.FailCalls
		returnNode.Usecs += tmpNode.Usecs
		returnNode.FailUsecs += tmpNode.FailUsecs
	}

	return returnNode
}

func PrintRedisDataMap(dataMap *RedisDataMap) {
	GLogger.Printf("-------------- RedisDataMap ------------------\n")
	GLogger.Printf("TimeInterval:%d\n", dataMap.TimeInterval)
	GLogger.Printf("StartTime:%s\n", dataMap.StartTime)
	GLogger.Printf("EndTime:%s\n", dataMap.EndTime)
	GLogger.Printf("all Calls:%d\n", dataMap.Calls)
	GLogger.Printf("all FailCalls:%d\n\n", dataMap.FailCalls)
	for key, node := range dataMap.Datas {
		GLogger.Printf("key:%s\n", key)
		GLogger.Printf("StartTime:%s\n", node.StartTime)
		GLogger.Printf("ProxyAddr:%s\n", node.ProxyAddr)
		GLogger.Printf("Calls:%d\n", node.Calls)
		GLogger.Printf("FailCalls:%d\n\n", node.FailCalls)
	}
}

func GetRedisDayData(redisDataFile string, interval int) *RedisDataMap {
	file, err := os.OpenFile(redisDataFile, os.O_RDONLY, os.ModePerm)
	if err != nil {
		GLogger.Printf("open file [%s] failed,err:%s\n", redisDataFile, err.Error())
		return nil
	}
	defer file.Close()

	var firstLine string
	var lastLine string
	var firstNode RedisDataMap = RedisDataMap{Datas: make(map[string]*RedisPerDataNode)}
	var lastNode RedisDataMap = RedisDataMap{Datas: make(map[string]*RedisPerDataNode)}
	var returnNode *RedisDataMap
	var flag bool = false
	var firstFlag bool = false
	var firstTimeStr string
	rd := bufio.NewReader(file)
	// Get first all data
	for {
		lineData, err := rd.ReadString('\n')
		if err != nil || err == io.EOF {
			break
		}
		if flag == false {
			firstLine = lineData
			// parse firstLine
			tmpNode := &RedisPerDataNode{}
			n1, err1 := fmt.Sscanf(firstLine, "%s\t%s\t%d\t%d\n",
				&tmpNode.StartTime, &tmpNode.ProxyAddr,
				&tmpNode.Calls, &tmpNode.FailCalls)
			if n1 != 4 {
				GLogger.Printf("sscanf form [%s] failed,err:%s\n", firstLine, err1.Error())
				continue
			}
			if firstFlag == false {
				firstTimeStr = tmpNode.StartTime
				firstFlag = true
			}
			if strings.Contains(lineData, firstTimeStr) == false {
				flag = true
			} else {
				//GLogger.Printf("firstLine:%s\n", lineData)
				firstNode.Datas[tmpNode.ProxyAddr] = tmpNode
			}
		} else {
			lastLine = lineData
		}
	}
	//GLogger.Printf("lastLine is:%s\n", lastLine)
	var lastTimeStr string
	n2, err2 := fmt.Sscanf(lastLine, "%s\t%*s\t%*s\t%*s\n", &lastTimeStr)
	if n2 != 1 {
		GLogger.Printf("Sscanf [%s] failed,err:%s\n", lastLine, err2.Error())
		return nil
	}
	// Get last all data
	file.Seek(0, os.SEEK_SET)
	rd = bufio.NewReader(file)
	for {
		lineData, err := rd.ReadString('\n')
		if err != nil || err == io.EOF {
			break
		}
		if strings.Contains(lineData, lastTimeStr) {
			tmpNode := &RedisPerDataNode{}
			n3, err3 := fmt.Sscanf(lineData, "%s\t%s\t%d\t%d\n",
				&tmpNode.StartTime, &tmpNode.ProxyAddr,
				&tmpNode.Calls, &tmpNode.FailCalls)
			if n3 != 4 {
				GLogger.Printf("sscanf form [%s] failed,err:%s\n", lineData, err3.Error())
				continue
			}
			lastNode.Datas[tmpNode.ProxyAddr] = tmpNode
		}

	}
	// calc oneday's data
	returnNode = &RedisDataMap{
		TimeInterval: interval,
		StartTime:    firstTimeStr,
		EndTime:      lastTimeStr,
		Datas:        make(map[string]*RedisPerDataNode),
	}
	for key, node := range lastNode.Datas {
		tmpNode := &RedisPerDataNode{
			ProxyAddr: key,
		}
		tmpBaseNode := firstNode.Datas[key]
		if tmpBaseNode != nil {
			tmpNode.Calls = node.Calls - tmpBaseNode.Calls
			tmpNode.FailCalls = node.FailCalls - tmpBaseNode.FailCalls
		} else {
			tmpNode.Calls = node.Calls
			tmpNode.FailCalls = node.FailCalls
		}
		returnNode.Datas[key] = tmpNode
		returnNode.Calls += tmpNode.Calls
		returnNode.FailCalls += tmpNode.FailCalls
	}
	//PrintRedisDataMap(returnNode)
	return returnNode
}

func PrintRedisCmdMap(cmd *RedisCmdMap) {
	GLogger.Printf("-------------Begin Redis Cmd Map-------------\n")
	GLogger.Printf("TimeInterval:%d\n", cmd.TimeInterval)
	GLogger.Printf("StartTime:%s\n", cmd.StartTime)
	GLogger.Printf("EndTime:%s\n", cmd.EndTime)
	GLogger.Printf("Calls:%d\n", cmd.Calls)
	GLogger.Printf("FailCalls:%d\n", cmd.FailCalls)
	GLogger.Printf("Usecs:%d\n", cmd.Usecs)
	GLogger.Printf("FailUsecs:%d\n\n", cmd.FailUsecs)

	for proxyaddr, proxy := range cmd.Proxys {
		GLogger.Printf("\tproxyaddr:%s\n", proxyaddr)
		GLogger.Printf("\tProxyAddr:%s\n", proxy.ProxyAddr)
		GLogger.Printf("\tCalls:%d\n", cmd.Calls)
		GLogger.Printf("\tFailCalls:%d\n", cmd.FailCalls)
		GLogger.Printf("\tUsecs:%d\n", cmd.Usecs)
		GLogger.Printf("\tFailUsecs:%d\n\n", cmd.FailUsecs)

		for _, data := range proxy.Cmds {
			GLogger.Printf("\t\tProxyAddr:%s\n", data.ProxyAddr)
			GLogger.Printf("\t\tCmd:%s\n", data.Cmd)
			GLogger.Printf("\t\tCalls:%d\n", data.Calls)
			GLogger.Printf("\t\tFailCalls:%d\n", data.FailCalls)
			GLogger.Printf("\t\tUsecs:%d\n", data.Usecs)
			GLogger.Printf("\t\tFailUsecs:%d\n\n", data.FailUsecs)
		}
	}
	GLogger.Printf("-------------End Redis Cmd Map-------------\n")
}

func GetRedisDayCmd(redisCmdFile string, interval int) *RedisCmdMap {
	file, err := os.OpenFile(redisCmdFile, os.O_RDONLY, os.ModePerm)
	if err != nil {
		GLogger.Printf("open file [%s] failed,err:%s\n", redisCmdFile, err.Error())
		return nil
	}
	defer file.Close()

	var firstLine string
	var lastLine string
	var firstNode RedisCmdMap = RedisCmdMap{Proxys: make(map[string]*RedisCmdProxy)}
	var lastNode RedisCmdMap = RedisCmdMap{Proxys: make(map[string]*RedisCmdProxy)}
	var returnNode *RedisCmdMap
	var flag bool = false
	var firstFlag bool = false
	var firstTimeStr string
	rd := bufio.NewReader(file)
	// Get first all data
	for {
		lineData, err := rd.ReadString('\n')
		if err != nil || err == io.EOF {
			break
		}
		if flag == false {
			firstLine = lineData
			// parse firstLine
			tmpNode := &RedisPerCmdNode{}
			n1, err1 := fmt.Sscanf(firstLine, "%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\n",
				&tmpNode.StartTime, &tmpNode.ProxyAddr, &tmpNode.Cmd, &tmpNode.Calls,
				&tmpNode.FailCalls, &tmpNode.FailUsecs, &tmpNode.Usecs, &tmpNode.UsecsPerCall)
			if n1 != 8 {
				GLogger.Printf("sscanf form [%s] failed,err:%s\n", firstLine, err1.Error())
				continue
			}
			if firstFlag == false {
				firstTimeStr = tmpNode.StartTime
				firstFlag = true
			}
			if strings.Contains(lineData, firstTimeStr) == false {
				flag = true
			} else {
				//GLogger.Printf("firstLine:%s\n", lineData)
				CmdProxy := firstNode.Proxys[tmpNode.ProxyAddr]
				if CmdProxy == nil {
					CmdProxy := &RedisCmdProxy{ProxyAddr: tmpNode.ProxyAddr, Cmds: make(map[string]*RedisPerCmdNode)}
					firstNode.Proxys[tmpNode.ProxyAddr] = CmdProxy
					CmdProxy.Cmds[tmpNode.Cmd] = tmpNode
				} else {
					CmdProxy.Cmds[tmpNode.Cmd] = tmpNode
				}
			}
		} else {
			lastLine = lineData
		}
	}
	//GLogger.Printf("lastLine is:%s\n", lastLine)
	var lastTimeStr string
	n2, err2 := fmt.Sscanf(lastLine, "%s\t%*s\t%*s\t%*s\t%*s\t%*s\t%*s\t%*s\n", &lastTimeStr)
	if n2 != 1 {
		GLogger.Printf("Sscanf [%s] failed,err:%s\n", lastLine, err2.Error())
		return nil
	}
	// Get last all data
	file.Seek(0, os.SEEK_SET)
	rd = bufio.NewReader(file)
	for {
		lineData, err := rd.ReadString('\n')
		if err != nil || err == io.EOF {
			break
		}
		if strings.Contains(lineData, lastTimeStr) {
			tmpNode := &RedisPerCmdNode{}
			n3, err3 := fmt.Sscanf(lineData, "%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\n",
				&tmpNode.StartTime, &tmpNode.ProxyAddr, &tmpNode.Cmd, &tmpNode.Calls,
				&tmpNode.FailCalls, &tmpNode.FailUsecs, &tmpNode.Usecs, &tmpNode.UsecsPerCall)
			if n3 != 8 {
				GLogger.Printf("sscanf form [%s] failed,err:%s\n", lineData, err3.Error())
				continue
			}
			CmdProxy := lastNode.Proxys[tmpNode.ProxyAddr]
			if CmdProxy == nil {
				CmdProxy := &RedisCmdProxy{ProxyAddr: tmpNode.ProxyAddr, Cmds: make(map[string]*RedisPerCmdNode)}
				lastNode.Proxys[tmpNode.ProxyAddr] = CmdProxy
				CmdProxy.Cmds[tmpNode.Cmd] = tmpNode
			} else {
				CmdProxy.Cmds[tmpNode.Cmd] = tmpNode
			}
		}

	}
	// calc oneday's data
	returnNode = &RedisCmdMap{
		TimeInterval: interval,
		StartTime:    firstTimeStr,
		EndTime:      lastTimeStr,
		Proxys:       make(map[string]*RedisCmdProxy),
	}
	for proxyaddr, proxy := range lastNode.Proxys {
		tmpProxy := returnNode.Proxys[proxyaddr]
		if tmpProxy == nil {
			tmpProxy = &RedisCmdProxy{ProxyAddr: proxyaddr, Cmds: make(map[string]*RedisPerCmdNode)}
			returnNode.Proxys[proxyaddr] = tmpProxy
		}

		for cmd, data := range proxy.Cmds {
			tmpNode := tmpProxy.Cmds[cmd]
			if tmpNode == nil {
				tmpNode = &RedisPerCmdNode{ProxyAddr: proxyaddr, Cmd: cmd}
				tmpProxy.Cmds[cmd] = tmpNode
			}
			tmpBaseProxy := firstNode.Proxys[proxyaddr]
			if tmpBaseProxy == nil {
				tmpNode.Calls = data.Calls
				tmpNode.FailCalls = data.FailCalls
				tmpNode.Usecs = data.Usecs
				tmpNode.FailCalls = data.FailCalls

				tmpProxy.Calls += data.Calls
				tmpProxy.FailCalls += data.FailCalls
				tmpProxy.Usecs += data.Usecs
				tmpProxy.FailUsecs += data.FailUsecs
			} else {
				tmpBaseCmd := tmpBaseProxy.Cmds[cmd]
				if tmpBaseCmd == nil {
					tmpNode.Calls = data.Calls
					tmpNode.FailCalls = data.FailCalls
					tmpNode.Usecs = data.Usecs
					tmpNode.FailCalls = data.FailCalls
				} else {
					tmpNode.Calls = data.Calls - tmpBaseCmd.Calls
					tmpNode.FailCalls = data.FailCalls - tmpBaseCmd.FailCalls
					tmpNode.Usecs = data.Usecs - tmpBaseCmd.Usecs
					tmpNode.FailUsecs = data.FailUsecs - tmpBaseCmd.FailUsecs
				}
				tmpProxy.Calls += tmpNode.Calls
				tmpProxy.FailCalls += tmpNode.FailCalls
				tmpProxy.Usecs += tmpNode.Usecs
				tmpProxy.FailUsecs += tmpNode.FailUsecs
			}
		}
		returnNode.Calls += tmpProxy.Calls
		returnNode.FailCalls += tmpProxy.FailCalls
		returnNode.Usecs += tmpProxy.Usecs
		returnNode.FailUsecs += tmpProxy.FailUsecs
	}
	return returnNode
}
