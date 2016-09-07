#!/bin/bash
PROG_DIR=/home/wangchunyan/work/go/myowncode/scripts/MoniRedisMem
LOG_DIR=/letv/wangchunyan/CodisMoni
LOGFILE=$LOG_DIR/redis_memory_monitor.log
LOGSIZE=104857600

PROG_MAIL=$PROG_DIR/http_mail.py
TMPMAIL=/tmp/mail.tmp

function log()
{
    echo "`date`: $1 $2 $3 $4 $5 $6 $7 $8" >> $LOGFILE
    # if logfile is too big,backup one
    filesize=`stat $LOGFILE | grep Size | awk '{print $2}'`
    if [ $filesize -gt $LOGSIZE ] ; then
        mv $LOGFILE $LOGFILE".old"
    fi
}

function debug()
{
#    log $1 $2 $3 $4 $5 $6 $7 $8
    return
}

REDISFILE=$PROG_DIR/redis.txt
REDISCLI=/home/wangchunyan/install/codis/bin/redis-cli

HTML_HEAD="<html><head><title>redis memory monitor(`date +"%x %X"`)</title></head><body><table border='1' cellspacing='0' cellpadding='0' align='center'><tr><th width='200'>Redis</th><th width='150'>Memory</th></tr>"

HTML_TAIL="</table></body></html>"

ALL_MEM=0
ALL_CONTENT=""

MAIL_ADDR="wangchunyan1@letv.com"
function send_mail()
{
	log "send_mail for monitor redis memory"
	mailtitle="REDIS内存实时统计(`date +"%x %X"`)"

	ALL_CONTENT=$ALL_CONTENT"<tr><td>ALL</td><td>`echo "$ALL_MEM  / 1048576" | bc` M</td></tr>"
	echo "`date` :send monitor data to:$MAIL_ADDR" >> $LOGFILE
	
	echo "$HTML_HEAD" >> $LOGFILE
	echo "$ALL_CONTENT" >> $LOGFILE
	echo "$HTML_TAIL" >> $LOGFILE
	
	echo "$HTML_HEAD" > $TMPMAIL
	echo "$ALL_CONTENT" >> $TMPMAIL
	echo "$HTML_TAIL" >> $TMPMAIL

	log "$PROG_MAIL push_redis_monitor \"$mailtitle\" $TMPMAIL"	
	$PROG_MAIL push_redis_monitor "$mailtitle" $TMPMAIL 1>/dev/null 2>/dev/null
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
	log "begin get_redis_mem"
	while read line
	do
		if [ -n "$line" ] ; then
			get_per_redis $line
		fi
	done < $REDISFILE
	log "end get_redis_mem"
}

get_redis_mem
send_mail

