#!/bin/bash
REDCLI=/letv/codis/bin/redis-cli
CONTENT=""
SERVER_GROUP=/tmp/server.group.checkmem
LOGFILE=`pwd`/moni_redis_memory.log

function dlog()
{
	echo "`date`:$1 $2 $3" >> $LOGFILE
}

function send_mail()
{
	mailtitle="REDIS内存过大告警(`date`)"
	subject="`echo $mailtitle |base64`"
	from="RedisMoni@letv.com"
	maillist="wangchunyan1@le.com,guozequn@le.com"

	(
	echo "Subject: =?UTF-8?B?$subject?="
	echo "MIME-Version: 1.0"
	echo "Content-Type: text/html;charset=\"UTF-8\""
	echo "Content-Disposition: inline"
	echo "To:$maillist"

	echo "<html><head><title>$subject</title></head><body>"
	echo "$CONTENT"
	echo "</body></html>"
	) | /usr/sbin/sendmail -f $from -t -i $maillist

	dlog "send mail ($mailtitle) to ($maillist),content($CONTENT)"
}

function check_redis()
{
	addr="$1"
	port="$2"
	maxnum=12.0

	res=`$REDCLI -h $addr -p $port info | grep "used_memory_human:" | awk -F: '{print $2}'`
	if [ `echo "$res" | grep "G"` != "" ] ; then
		num=`echo "$res" | awk -FG '{print $1}'`
		if [ $(echo "$num > $maxnum"|bc) -eq 1 ] ; then
			dlog "server($addr:$port) memory is big than $maxnum G"
			CONTENT="$CONTENT server($addr:$port),memory($num G),big than maxnum($maxnum G),should migrate redis.<br />"
		fi
	fi
}

function check_memory()
{
	CONTENT=""
	while read line
	do
		flag=`echo "$line" | grep "master"`
		if [ "$flag" != "" ] ; then
			master=`echo "$line" | awk -F' ' '{print $1}'`
			master_addr=`echo "$master" | awk -F: '{print $1}'`
			master_port=`echo "$master" | awk -F: '{print $2}'`
			check_redis "$master_addr" "$master_port"
		fi
	done <$SERVER_GROUP

	if [ "$CONTENT" != "" ] ; then
		send_mail
	fi
}

CURDIR=`pwd`
PROG=$0
function main()
{
	dlog "$PROG start..."
	while [ 1 ]
	do
		$CURDIR/getServerGroup.sh
		check_memory
		sleep 3600
	done
}


main
