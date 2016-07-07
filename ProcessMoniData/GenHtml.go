package main

import (
	//"bufio"
	"fmt"
	//"io"
	"strconv"
	"strings"
	"time"
)

var FirstBlock string = `
<html>
<head>
    <meta charset="utf-8">
    <style type="text/css">
        .ui-table{
            width:1200px;
            padding:0;
            margin: 0 auto;
        }
        .ui-table,.ui-table td, .ui-table th {
            border-collapse: collapse;
            text-align: left;
            border: 1px solid #E7E7E7;
            padding: 7px;
        }
        .ui-table td, .m-table td, .ui-table th, .m-table th {
            min-width: 50px;
        }
        .ui-table thead th {
            text-align: center;
        }
        .ui-table th {
            background: #787878 none repeat scroll 0% 0% ;
            color : #FFF;
        }
        .ui-table table, .ui-table td, .ui-table th {
            text-align: left;
            border: 1px solid #E7E7E7;
            padding: 7px;
        }
        .ui-table tr:nth-child(2n) {
            background: #F9F9F9 none repeat scroll 0% 0%;
            margin: 10px;
        }
    </style>
</head>
    <body>
`

var EndBlock string = `
    </body>
</html>
`

var TableEndStr string = `
        </table>
	<hr>
`
var ProxyGlobalHead string = `
<h3>代理全局信息统计</h3>
    <table class="ui-table">
        <th>代理地址</th>
	<th>总连接</th>
	<th>处理失败连接</th>
	<th>总操作</th>
	<th>OPS</th>
	<th>处理失败操作</th>
	<th>处理成功操作</th>
`

var BgRedColor string = `bgcolor="red"`
var ProxyDataTemp string = `
<tr>
    <td>{proxy-addr}</td>
    <td>{conn-num}</td>
    <td {conn-color}>{conn-fail}</td>
    <td>{op-num}</td>
    <td>{proxy-ops}</td>
    <td {ops-color}>{op-fail}</td>
    <td>{op-succ}</td>
</tr>
`

func GenProxyDataHtmlPer(data *ProxyPerDataNode) string {
	tmpaddr := GetIpAddrByUrl(data.Addr)
	if len(tmpaddr) == 0 {
		GLogger.Printf("GenProxyDataHtmlPer get proxy addr from [%s] failed", data.Addr)
		return ""
	}
	var e error
	var start, end int
	start, e = strconv.Atoi(data.StartTime)
	if e != nil {
		GLogger.Printf("Atoi starttime [%s] failed", data.StartTime)
		return ""
	}
	end, e = strconv.Atoi(data.EndTime)
	if e != nil {
		GLogger.Printf("Atoi endtime [%s] failed", data.StartTime)
		return ""
	}
	secs := (end - start)
	s := ProxyDataTemp
	s = strings.Replace(s, "{proxy-addr}", tmpaddr, 1)
	s = strings.Replace(s, "{conn-num}", strconv.FormatInt(data.ConnNum, 10), 1)
	s = strings.Replace(s, "{conn-fail}", strconv.FormatInt(data.ConnFailNum, 10), 1)

	if float64(data.ConnFailNum)/float64(data.ConnNum) > 0.01 {
		s = strings.Replace(s, "{conn-color}", BgRedColor, 1)
	} else {
		s = strings.Replace(s, "{conn-color}", "", 1)
	}

	s = strings.Replace(s, "{op-num}", strconv.FormatInt(data.OpNum, 10), 1)
	if secs > 0 {
		s = strings.Replace(s, "{proxy-ops}", strconv.FormatInt(data.OpNum/int64(secs), 10), 1)
	} else {
		s = strings.Replace(s, "{proxy-ops}", "---", 1)
	}

	s = strings.Replace(s, "{op-fail}", strconv.FormatInt(data.OpFailNum, 10), 1)
	if data.OpNum > 0 && float64(data.OpFailNum)/float64(data.OpNum) > 0.0002 {
		s = strings.Replace(s, "{ops-color}", BgRedColor, 1)
	} else {
		s = strings.Replace(s, "{ops-color}", "", 1)
	}

	s = strings.Replace(s, "{op-succ}", strconv.FormatInt(data.OpSuccNum, 10), 1)
	//GLogger.Printf("s is :%s", s)
	return s
}

var s string = ""

func GenProxyDataHtml2(data [64]*ProxyPerDataNode) string {
	var proxydata string = ""
	for _, proxy := range data {
		if proxy == nil {
			continue
		}
		if proxy.OpNum <= 10 {
			continue
		}
		s = GenProxyDataHtmlPer(proxy)
		proxydata += s
		//GLogger.Printf("tmpproxydata is:%s", proxydata)
	}
	//GLogger.Printf("proxydata is:\n%s\n", proxydata)
	return proxydata
}

func GenProxyDataHtml(proxyAddr string, data *ProxyPerDataNode) string {
	stFmt := "2006-01-02 15:04:05"
	start, _ := time.Parse(stFmt, strings.Replace(data.StartTime, ".", " ", 1))
	end, _ := time.Parse(stFmt, strings.Replace(data.EndTime, ".", " ", 1))
	secs := end.Unix() - start.Unix()

	s := ProxyDataTemp
	s = strings.Replace(s, "{proxy-addr}", proxyAddr, 1)
	s = strings.Replace(s, "{conn-num}", strconv.FormatInt(data.ConnNum, 10), 1)
	s = strings.Replace(s, "{conn-fail}", strconv.FormatInt(data.ConnFailNum, 10), 1)

	if float64(data.ConnFailNum)/float64(data.ConnNum) > 0.01 {
		s = strings.Replace(s, "{conn-color}", BgRedColor, 1)
	} else {
		s = strings.Replace(s, "{conn-color}", "", 1)
	}

	s = strings.Replace(s, "{op-num}", strconv.FormatInt(data.OpNum, 10), 1)
	if secs > 0 {
		s = strings.Replace(s, "{proxy-ops}", strconv.FormatInt(data.OpNum/secs, 10), 1)
	} else {
		s = strings.Replace(s, "{proxy-ops}", "---", 1)
	}

	s = strings.Replace(s, "{op-fail}", strconv.FormatInt(data.OpFailNum, 10), 1)
	if data.OpNum > 0 && float64(data.OpFailNum)/float64(data.OpNum) > 0.0002 {
		s = strings.Replace(s, "{ops-color}", BgRedColor, 1)
	} else {
		s = strings.Replace(s, "{ops-color}", "", 1)
	}

	s = strings.Replace(s, "{op-succ}", strconv.FormatInt(data.OpSuccNum, 10), 1)
	return s
}

var ProxyCmdHead string = `
<h3>代理操作统计</h3>
    <table class="ui-table">
        <th>代理地址</th>
	<th>操作类型</th>
	<th>操作次数</th>
	<th>操作耗时</th>
	<th>失败次数</th>
	<th>失败耗时</th>
`
var ProxyCmdTemp string = `
<tr>
    <td>{proxy-addr}</td>
    <td>{cmd-type}</td>
    <td>{op-num}</td>
    <td>{op-sec}</td>
    <td>{op-fail}</td>
    <td>{fail-sec}</td>
</tr>
`

func GenProxyCmdHtml(proxyAddr string, cmd *ProxyCmdMap) string {
	var data string = ""
	/*s := ProxyCmdTemp
	s = strings.Replace(s, "{proxy-addr}", "ALL", 1)
	s = strings.Replace(s, "{cmd-type}", "ALL", 1)
	s = strings.Replace(s, "{op-num}", strconv.FormatInt(cmd.Calls, 10), 1)
	s = strings.Replace(s, "{op-sec}", strconv.FormatInt(cmd.Usecs, 10), 1)
	s = strings.Replace(s, "{op-fail}", strconv.FormatInt(cmd.FailCalls, 10), 1)
	s = strings.Replace(s, "{fail-sec}", strconv.FormatInt(cmd.FailUsecs, 10), 1)

	data += s*/

	for key, cmds := range cmd.Cmds {
		s = ProxyCmdTemp
		s = strings.Replace(s, "{proxy-addr}", proxyAddr, 1)
		s = strings.Replace(s, "{cmd-type}", key, 1)
		s = strings.Replace(s, "{op-num}", strconv.FormatInt(cmds.Calls, 10), 1)
		s = strings.Replace(s, "{op-sec}", strconv.FormatInt(cmds.Usecs, 10), 1)
		s = strings.Replace(s, "{op-fail}", strconv.FormatInt(cmds.FailCalls, 10), 1)
		s = strings.Replace(s, "{fail-sec}", strconv.FormatInt(cmds.FailUsecs, 10), 1)
		data += s
	}
	return data
}

var RedisDataHead string = `
<h3>Redis全局信息统计</h3>
    <table class="ui-table">
    <th>Redis地址</th>
    <th>代理地址</th>
    <th>总操作次数</th>
    <th>操作失败次数</th>
`
var RedisDataTemp string = `
<tr>
    <td>{redis-addr}</td>
    <td>{proxy-addr}</td>
    <td>{op-num}</td>
    <td>{op-fail}</td>
</tr>
`

func GenRedisDataHtml(redisAddr string, cmd *RedisDataMap) string {
	var data string = ""
	/*s := RedisDataTemp
	s = strings.Replace(s, "{redis-addr}", "ALL", 1)
	s = strings.Replace(s, "{proxy-addr}", "ALL", 1)
	s = strings.Replace(s, "{op-num}", strconv.FormatInt(cmd.Calls, 10), 1)
	s = strings.Replace(s, "{op-fail}", strconv.FormatInt(cmd.FailCalls, 10), 1)
	data += s*/

	for _, cmds := range cmd.Datas {
		s = RedisDataTemp
		s = strings.Replace(s, "{redis-addr}", redisAddr, 1)
		s = strings.Replace(s, "{proxy-addr}", cmds.ProxyAddr, 1)
		s = strings.Replace(s, "{op-num}", strconv.FormatInt(cmd.Calls, 10), 1)
		s = strings.Replace(s, "{op-fail}", strconv.FormatInt(cmd.FailCalls, 10), 1)
		data += s
	}
	return data
}

var RedisSummaryHead string = `
<h3>Redis全局信息统计</h3>
<table class="ui-table">
    <th>Redis地址</th>
    <th>总操作次数</th>
    <th>OPS</th>
    <th>操作失败次数</th>
`
var RedisSummaryTemp string = `
<tr>
    <td>{redis-addr}</td>
    <td>{op-num}</td>
    <td>{redis-ops}</td>
    <td {op-color}>{op-fail}</td>
</tr>
`

func GenRedisSummaryHtml(redisAddr string, cmd *RedisDataMap) string {
	var s string = RedisSummaryTemp
	stFmt := "2006-01-02 15:04:05"
	start, _ := time.Parse(stFmt, strings.Replace(cmd.StartTime, ".", " ", 1))
	end, _ := time.Parse(stFmt, strings.Replace(cmd.EndTime, ".", " ", 1))
	secs := end.Unix() - start.Unix()

	s = strings.Replace(s, "{redis-addr}", redisAddr, 1)
	s = strings.Replace(s, "{op-num}", strconv.FormatInt(cmd.Calls, 10), 1)
	if secs > 0 {
		s = strings.Replace(s, "{redis-ops}", strconv.FormatInt(cmd.Calls/secs, 10), 1)
	} else {
		s = strings.Replace(s, "{redis-ops}", "---", 1)
	}
	s = strings.Replace(s, "{op-fail}", strconv.FormatInt(cmd.FailCalls, 10), 1)
	if cmd.Calls > 0 {
		if float64(cmd.FailCalls)/float64(cmd.Calls) > 0.0002 {
			s = strings.Replace(s, "{op-color}", BgRedColor, 1)
		} else {
			s = strings.Replace(s, "{op-color}", "", 1)
		}
	} else {
		s = strings.Replace(s, "{op-color}", "", 1)
	}
	return s
}

func GenRedisPerRecord(addr string, record *RedisDataRecord) string {
	// get record start and end time
	var start, end int
	var e error
	if record == nil {
		GLogger.Printf("record is nil")
		return ""
	}
	start, e = strconv.Atoi(record.StartTime)
	if e != nil {
		GLogger.Printf("atoi start time [%s] failed", record.StartTime)
		return ""
	}
	end, e = strconv.Atoi(record.EndTime)
	if e != nil {
		GLogger.Printf("atoi end time [%s] failed", record.EndTime)
		return ""
	}
	var s string = RedisSummaryTemp
	secs := (end - start)
	//GLogger.Printf("start[%s],end[%s],secs[%d]", record.StartTime, record.EndTime, secs)

	s = strings.Replace(s, "{redis-addr}", addr, 1)
	s = strings.Replace(s, "{op-num}", strconv.FormatInt(record.OpNum, 10), 1)
	if secs > 0 {
		s = strings.Replace(s, "{redis-ops}", strconv.FormatInt(record.OpNum/int64(secs), 10), 1)
	} else {
		s = strings.Replace(s, "{redis-ops}", "---", 1)
	}
	s = strings.Replace(s, "{op-fail}", strconv.FormatInt(record.OpFailNum, 10), 1)
	if record.OpNum > 0 {
		if float64(record.OpFailNum)/float64(record.OpNum) > 0.0002 {
			s = strings.Replace(s, "{op-color}", BgRedColor, 1)
		} else {
			s = strings.Replace(s, "{op-color}", "", 1)
		}
	} else {
		s = strings.Replace(s, "{op-color}", "", 1)
	}
	//GLogger.Printf("s is :%s", s)
	return s
}

func GenRedisData2(data *RedisDataStatistic) string {
	var datastr string = ""
	var allRecord RedisDataRecord
	for addr, redis := range data.Records {
		if len(addr) == 0 || redis == nil {
			continue
		}
		if redis.OpNum <= 0 {
			continue
		}
		allRecord.OpNum += redis.OpNum
		allRecord.OpFailNum += redis.OpFailNum
		allRecord.StartTime = redis.StartTime
		allRecord.EndTime = redis.EndTime
		s = GenRedisPerRecord(addr, redis)
		datastr += s
	}
	s = GenRedisPerRecord("ALL", &allRecord)
	datastr += s
	//GLogger.Printf("data str is:%s", datastr)
	return datastr
}

func GenRedisPerCmd(cmdname string, cmds *RedisCmdRecord) string {
	if cmds == nil {
		return ""
	}
	s := RedisCmdTemp
	s = strings.Replace(s, "{cmd-type}", cmdname, 1)
	s = strings.Replace(s, "{op-num}", strconv.FormatInt(cmds.OpNum, 10), 1)
	if cmds.OpNum != 0 {
		var opsec float64 = (float64(cmds.OpSecs) / float64(cmds.OpNum)) / 1000
		s = strings.Replace(s, "{op-sec}", fmt.Sprintf("%.6f", opsec), 1)
		if opsec > 150.0 {
			s = strings.Replace(s, "{op-color}", BgRedColor, 1)
		} else {
			s = strings.Replace(s, "{op-color}", "", 1)
		}
	} else {
		s = strings.Replace(s, "{op-sec}", "---", 1)
		s = strings.Replace(s, "{op-color}", "", 1)
	}
	s = strings.Replace(s, "{op-fail}", strconv.FormatInt(cmds.OpFailNum, 10), 1)
	if cmds.OpFailSecs > 0 {
		s = strings.Replace(s, "{fail-sec}", fmt.Sprintf("%.6f", (float64(cmds.OpFailSecs)/float64(cmds.OpFailNum))/1000), 1)
	} else {
		s = strings.Replace(s, "{fail-sec}", "---", 1)
	}
	return s
}

func GenRedisCmd2(data *RedisCmdStatistic) string {
	var datastr string = ""
	for cmdname, redis := range data.Cmds {
		if len(cmdname) == 0 || redis == nil {
			continue
		}
		if redis.OpNum <= 0 {
			continue
		}
		s = GenRedisPerCmd(cmdname, redis)
		datastr += s
	}
	return datastr
}

var RedisCmdHead string = `
<h3>Redis操作统计</h3>
<table class="ui-table">
    <th>操作类型</th>
    <th>操作次数</th>
    <th>单次操作耗时(毫秒)</th>
    <th>失败次数</th>
    <th>平均失败耗时(毫秒)</th>
`
var RedisCmdTemp string = `
<tr>
    <td>{cmd-type}</td>
    <td>{op-num}</td>
    <td {op-color}>{op-sec}</td>
    <td>{op-fail}</td>
    <td>{fail-sec}</td>
</tr>
`

var AllRedisCmdNodes map[string]*RedisPerCmdNode = make(map[string]*RedisPerCmdNode)

func InitAllRedisCmdNode() {
	for _, node := range AllRedisCmdNodes {
		node.Calls = 0
		node.FailCalls = 0
		node.Usecs = 0
		node.FailUsecs = 0
	}
}

func CalcRedisCmd(redisAddr string, cmd *RedisCmdMap) {
	for _, proxy := range cmd.Proxys {
		for cmd, cmds := range proxy.Cmds {
			tmpCmd := AllRedisCmdNodes[cmd]
			if tmpCmd == nil {
				tmpCmd = &RedisPerCmdNode{Cmd: cmd}
				AllRedisCmdNodes[cmd] = tmpCmd
			}
			tmpCmd.Calls += cmds.Calls
			tmpCmd.FailCalls += cmds.FailCalls
			tmpCmd.Usecs += cmds.Usecs
			tmpCmd.FailUsecs += cmds.FailUsecs
		}
	}
}

func GenRedisCmdHtml() string {
	var data string = ""
	var s string = ""
	for key, cmds := range AllRedisCmdNodes {
		s = RedisCmdTemp
		s = strings.Replace(s, "{cmd-type}", key, 1)
		s = strings.Replace(s, "{op-num}", strconv.FormatInt(cmds.Calls, 10), 1)
		if cmds.Calls != 0 {
			var opsec float64 = (float64(cmds.Usecs) / float64(cmds.Calls)) / 1000
			s = strings.Replace(s, "{op-sec}", fmt.Sprintf("%.6f", opsec), 1)
			if opsec > 100.0 {
				s = strings.Replace(s, "{op-color}", BgRedColor, 1)
			} else {
				s = strings.Replace(s, "{op-color}", "", 1)
			}
		} else {
			s = strings.Replace(s, "{op-sec}", "---", 1)
			s = strings.Replace(s, "{op-color}", "", 1)
		}
		s = strings.Replace(s, "{op-fail}", strconv.FormatInt(cmds.FailCalls, 10), 1)
		if cmds.FailUsecs > 0 {
			s = strings.Replace(s, "{fail-sec}", fmt.Sprintf("%.6f", (float64(cmds.FailUsecs)/float64(cmds.FailCalls))/1000), 1)
		} else {
			s = strings.Replace(s, "{fail-sec}", "---", 1)
		}
		data += s
	}
	return data
}

func GenDayReportHtml(proxyData string, proxyCmd string, redisData string, redisCmd string) string {
	var data string = ""
	data += FirstBlock

	data += ProxyGlobalHead
	data += proxyData
	data += TableEndStr

	data += RedisDataHead
	data += redisData
	data += TableEndStr

	data += ProxyCmdHead
	data += proxyCmd
	data += TableEndStr

	data += RedisCmdHead
	data += redisCmd
	data += TableEndStr

	data += EndBlock
	return data
}

func GenDaySummaryReportHtml(proxyData string, redisData string, redisCmd string) string {
	var data string = ""
	data += FirstBlock

	data += ProxyGlobalHead
	data += proxyData
	data += TableEndStr

	data += RedisSummaryHead
	data += redisData
	data += TableEndStr

	data += RedisCmdHead
	data += redisCmd
	data += TableEndStr

	data += EndBlock
	return data
}
