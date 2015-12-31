package main

import (
	//"bufio"
	//"fmt"
	//"io"
	//"os"
	"strconv"
	"strings"
)

var FirstBlock string = `<html>
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
    <body>`

var EndBlock string = `</body>
</html>`

var TableEndStr string = "</table><hr>"
var ProxyGlobalHead string = `<h3>代理全局信息统计</h3>
    <table class="ui-table">
        <th>代理地址</th><th>总连接</th><th>处理失败连接</th><th>总操作</th><th>处理失败操作</th><th>处理成功操作</th>`

var ProxyDataTemp string = "<tr><td>{proxy-addr}</td><td>{conn-num}</td><td>{conn-fail}</td><td>{op-num}</td><td>{op-fail}</td><td>{op-succ}</td></tr>"

func GenProxyDataHtml(proxyAddr string, data *ProxyPerDataNode) string {
	s := ProxyDataTemp
	s = strings.Replace(s, "{proxy-addr}", proxyAddr, -1)
	s = strings.Replace(s, "{conn-num}", strconv.FormatInt(data.ConnNum, 10), -1)
	s = strings.Replace(s, "{conn-fail}", strconv.FormatInt(data.ConnFailNum, 10), -1)
	s = strings.Replace(s, "{op-num}", strconv.FormatInt(data.OpNum, 10), -1)
	s = strings.Replace(s, "{op-fail}", strconv.FormatInt(data.OpFailNum, 10), -1)
	s = strings.Replace(s, "{op-succ}", strconv.FormatInt(data.OpSuccNum, 10), -1)
	return s
}

var ProxyCmdHead string = `<h3>代理操作统计</h3>
    <table class="ui-table">
        <th>代理地址</th><th>操作类型</th><th>操作次数</th><th>操作耗时</th><th>失败次数</th><th>失败耗时</th>`
var ProxyCmdTemp string = "<tr><td>{proxy-addr}</td><td>{cmd-type}</td><td>{op-num}</td><td>{op-sec}</td><td>{op-fail}</td><td>{fail-sec}</td></tr>"

func GenProxyCmdHtml(proxyAddr string, cmd *ProxyCmdMap) string {
	var data string = ""
	var s string = ""
	/*s := ProxyCmdTemp
	s = strings.Replace(s, "{proxy-addr}", "ALL", -1)
	s = strings.Replace(s, "{cmd-type}", "ALL", -1)
	s = strings.Replace(s, "{op-num}", strconv.FormatInt(cmd.Calls, 10), -1)
	s = strings.Replace(s, "{op-sec}", strconv.FormatInt(cmd.Usecs, 10), -1)
	s = strings.Replace(s, "{op-fail}", strconv.FormatInt(cmd.FailCalls, 10), -1)
	s = strings.Replace(s, "{fail-sec}", strconv.FormatInt(cmd.FailUsecs, 10), -1)

	data += s*/

	for key, cmds := range cmd.Cmds {
		s = ProxyCmdTemp
		s = strings.Replace(s, "{proxy-addr}", proxyAddr, -1)
		s = strings.Replace(s, "{cmd-type}", key, -1)
		s = strings.Replace(s, "{op-num}", strconv.FormatInt(cmds.Calls, 10), -1)
		s = strings.Replace(s, "{op-sec}", strconv.FormatInt(cmds.Usecs, 10), -1)
		s = strings.Replace(s, "{op-fail}", strconv.FormatInt(cmds.FailCalls, 10), -1)
		s = strings.Replace(s, "{fail-sec}", strconv.FormatInt(cmds.FailUsecs, 10), -1)
		data += s
	}
	return data
}

var RedisDataHead string = `<h3>Redis全局信息统计</h3>
    <table class="ui-table">
        <th>Redis地址</th><th>代理地址</th><th>总操作次数</th><th>操作失败次数</th>`
var RedisDataTemp string = "<tr><td>{redis-addr}</td><td>{proxy-addr}</td><td>{op-num}</td><td>{op-fail}</td>"

func GenRedisDataHtml(redisAddr string, cmd *RedisDataMap) string {
	var data string = ""
	var s string = ""
	/*s := RedisDataTemp
	s = strings.Replace(s, "{redis-addr}", "ALL", -1)
	s = strings.Replace(s, "{proxy-addr}", "ALL", -1)
	s = strings.Replace(s, "{op-num}", strconv.FormatInt(cmd.Calls, 10), -1)
	s = strings.Replace(s, "{op-fail}", strconv.FormatInt(cmd.FailCalls, 10), -1)
	data += s*/

	for _, cmds := range cmd.Datas {
		s = RedisDataTemp
		s = strings.Replace(s, "{redis-addr}", redisAddr, -1)
		s = strings.Replace(s, "{proxy-addr}", cmds.ProxyAddr, -1)
		s = strings.Replace(s, "{op-num}", strconv.FormatInt(cmd.Calls, 10), -1)
		s = strings.Replace(s, "{op-fail}", strconv.FormatInt(cmd.FailCalls, 10), -1)
		data += s
	}
	return data
}

var RedisCmdHead string = `<h3>Redis操作统计</h3>
    <table class="ui-table">
        <th>Redis地址</th><th>代理地址</th><th>操作类型</th><th>操作次数</th><th>操作耗时</th><th>失败次数</th><th>失败耗时</th>`
var RedisCmdTemp string = "<tr><td>{redis-addr}</td><td>{proxy-addr}</td><td>{cmd-type}</td><td>{op-num}</td><td>{op-sec}</td><td>{op-fail}</td><td>{fail-sec}</td></tr>"

func GenRedisCmdHtml(redisAddr string, cmd *RedisCmdMap) string {
	var data string = ""
	var s string = ""
	/*s := RedisCmdTemp
	s = strings.Replace(s, "{redis-addr}", "ALL", -1)
	s = strings.Replace(s, "{proxy-addr}", "ALL", -1)
	s = strings.Replace(s, "{cmd-type}", "ALL", -1)
	s = strings.Replace(s, "{op-num}", strconv.FormatInt(cmd.Calls, 10), -1)
	s = strings.Replace(s, "{op-sec}", strconv.FormatInt(cmd.Usecs, 10), -1)
	s = strings.Replace(s, "{op-fail}", strconv.FormatInt(cmd.FailCalls, 10), -1)
	s = strings.Replace(s, "{fail-sec}", strconv.FormatInt(cmd.FailUsecs, 10), -1)
	data += s*/

	for _, proxy := range cmd.Proxys {
		for _, cmds := range proxy.Cmds {
			s = RedisCmdTemp
			s = strings.Replace(s, "{redis-addr}", redisAddr, -1)
			s = strings.Replace(s, "{proxy-addr}", proxy.ProxyAddr, -1)
			s = strings.Replace(s, "{cmd-type}", cmds.Cmd, -1)
			s = strings.Replace(s, "{op-num}", strconv.FormatInt(cmds.Calls, 10), -1)
			s = strings.Replace(s, "{op-sec}", strconv.FormatInt(cmds.Usecs, 10), -1)
			s = strings.Replace(s, "{op-fail}", strconv.FormatInt(cmds.FailCalls, 10), -1)
			s = strings.Replace(s, "{fail-sec}", strconv.FormatInt(cmds.FailUsecs, 10), -1)
			data += s
		}
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
