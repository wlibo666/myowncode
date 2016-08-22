#!/bin/bash
REDISFILE=/home/wangchunyan/work/go/myowncode/scripts/MoniRedisMem/redis.txt
REDISCLI=/home/wangchunyan/install/codis/bin/redis-cli

HTML_HEAD="<html><head><title>redis memory monitor(`date +"%x %X"`)</title></head><body><table border='1' cellspacing='0' cellpadding='0' align='center'><tr><th width='200'>Redis</th><th width='150'>Memory</th></tr>"

HTML_TAIL="</table></body></html>"

ALL_MEM=0
ALL_CONTENT=""

MAIL_ADDR="wangchunyan1@letv.com"
function send_mail()
{
	mailtitle="REDIS内存实时统计(`date +"%x %X"`)"
	subject="`echo $mailtitle |base64`"

	ALL_CONTENT=$ALL_CONTENT"<tr><td>ALL</td><td>`echo "$ALL_MEM  / 1048576" | bc` M</td></tr>"
	echo "send monitor data to:$MAIL_ADDR" >> /tmp/redismonitor.txt
	echo "$ALL_CONTENT" >> /tmp/redismonitor.txt
	(
	echo "Subject: =?UTF-8?B?$subject?="
	echo "MIME-Version: 1.0"
	echo "Content-Type: text/html;charset=\"UTF-8\""
	echo "Content-Disposition: inline"
	echo "To:$MAIL_ADDR"

	echo "$HTML_HEAD"
	echo "$ALL_CONTENT"
	echo "$HTML_TAIL"
	) | /usr/sbin/sendmail -t -i $MAIL_ADDR
}

TMPFILE=/tmp/.number.swap
function get_per_redis()
{
	addr=`echo "$1" | awk -F: '{print $1}'`
	port=`echo "$1" | awk -F: '{print $2}'`

	mem=`$REDISCLI -h $addr -p $port info | grep used_memory: | awk -F: '{print $2}'`
	mem=`echo "$mem">$TMPFILE ; dos2unix $TMPFILE 1>/dev/null 2>/dev/null; cat $TMPFILE`
	
	ALL_MEM=`echo "$ALL_MEM + $mem" | bc`
	ALL_CONTENT=$ALL_CONTENT"<tr><td>$1</td><td>$mem</td></tr>"
}

function get_redis_mem()
{
	while read line
	do
		if [ -n "$line" ] ; then
			get_per_redis $line
		fi
	done < $REDISFILE
}

get_redis_mem
send_mail

