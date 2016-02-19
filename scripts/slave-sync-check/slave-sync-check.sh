#!/bin/bash

CODIS_CONFIG_PROG=/home/wangchunyan/install/codis/bin/codis-config
CODIS_CONFIG_FILE=/home/wangchunyan/install/codis/conf/config.ini
SERVER_GROUP=/tmp/server.group
REDIS_CLI=/home/wangchunyan/install/codis/bin/redis-cli
REDIS_MAX_DIFF_NUM=1000

CUR_DIR=`pwd`
CHECK_LOG=$CUR_DIR/slave_sync_check.log

tmpline=""
function record_info()
{
	addr=`echo "$1" | grep "addr" | awk -F'"' '{print $4}'`
	if [ "$addr" != "" ] ; then
		tmpline="$addr"
	fi

	group=`echo "$1" | grep "group_id" | awk -F':' '{print $2}' | awk -F',' '{print $1}'`
	if [ "$group" != "" ] ; then
		tmpline="$tmpline group-`echo "$group" | tr -d ' '`"
	fi
 
	role=`echo "$1" | grep "type" | awk -F'"' '{print $4}'`
	if [ "$role" != "" ] ; then
		tmpline="$tmpline $role"
		echo "$tmpline" >> $SERVER_GROUP
		tmpline=""
	fi 
}

function get_groups()
{
	$CODIS_CONFIG_PROG -c $CODIS_CONFIG_FILE server list > /tmp/servergroup.json
	rm $SERVER_GROUP 1>/dev/null 2>/dev/null
	beginflag=0
	endflag=0
	
	while read line
	do
		flag=`echo "$line" | grep "product_name"`
		if [ "$flag" != "" ] ; then
			beginflag=0
			endflag=0
		fi

		flag=`echo "$line" | grep "servers"`
		if [ "$flag" != "" ] && [ $endflag -eq 0 ] ; then
			beginflag=1
			tmpline=""
		fi

		flag=`echo "$line" | grep "]"`
		if [ "$flag" != "" ] && [ $beginflag -eq 1 ] ; then
			endflag=1
		fi

		# group information
		if [ $beginflag -eq 1 ] && [ $endflag -eq 0 ] ; then
			record_info "$line"
		fi

	done </tmp/servergroup.json
}

function check_redis()
{
	master_addr=`echo "$1" | awk -F: '{print $1}'`
	master_port=`echo "$1" | awk -F: '{print $2}'`
	slave_addr=`echo "$2" | awk -F: '{print $1}'`
	slave_port=`echo "$2" | awk -F: '{print $2}'`
	
	master_size=`$REDIS_CLI -h $master_addr -p $master_port dbsize | tr -d ' '`
	slave_size=`$REDIS_CLI -h $slave_addr -p $slave_port dbsize | tr -d ' '`

	#echo "master:$master_addr:$master_port size $master_size"
	#echo "slave:$slave_addr:$slave_port size $slave_size"
	
	diff_num=`expr $master_size - $slave_size`
	
	if [ $diff_num -ge $REDIS_MAX_DIFF_NUM ] ; then
		ret=`$REDIS_CLI -h $slave_addr -p $slave_port slaveof no one`
		echo "`date`: set $slave_addr:$slave_port slaveof no one,RET:$ret" >> $CHECK_LOG
		ret=`$REDIS_CLI -h $slave_addr -p $slave_port slaveof $master_addr $master_port`
		echo "`date`: set $slave_addr:$slave_port slaveof $master_addr $master_port,RET:$ret" >> $CHECK_LOG
	fi
}

function check_slave()
{
	while read line
	do
		flag=`echo "$line" | grep "slave"`
		if [ "$flag" != "" ] ; then
			slave=`echo "$line" | awk -F' ' '{print $1}'`
			group=`echo "$line" | awk -F' ' '{print $2}'`
			master=`cat $SERVER_GROUP | grep "$group master" | awk -F' ' '{print $1}'`
			if [ "$master" != "" ] ; then
				check_redis "$master" "$slave"
			fi 
		fi
	done <$SERVER_GROUP
}

function main()
{
	while [ 1 ]
	do
		get_groups
		check_slave
		sleep 1200
	done
}

main

