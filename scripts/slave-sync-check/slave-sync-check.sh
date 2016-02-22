#!/bin/bash

CONFIG_FILE=""

CODIS_CONFIG_PROG=""
CODIS_CONFIG_FILE=""
SERVER_GROUP=""
REDIS_CLI=""

REDIS_MAX_DIFF_NUM=1000

CUR_DIR=`pwd`
LOG_FILE=$CUR_DIR/slave_sync_check.log

function sh_log()
{
	echo "`date`: $1 $2 $3 $4" >> $LOG_FILE
}

function load_config()
{
	CODIS_CONFIG_PROG=`cat $CONFIG_FILE | grep "CODIS_CONFIG_PROG" | awk -F= '{print $2}'`
	CODIS_CONFIG_FILE=`cat $CONFIG_FILE | grep "CODIS_CONFIG_FILE" | awk -F= '{print $2}'`
	SERVER_GROUP=`cat $CONFIG_FILE | grep "SERVER_GROUP" | awk -F= '{print $2}'`
	SERVER_GROUP=$SERVER_GROUP.$$
	REDIS_CLI=`cat $CONFIG_FILE | grep "REDIS_CLI" | awk -F= '{print $2}'`
	errorflag=0

	if [ -z "$CODIS_CONFIG_PROG" ] ; then
		errorflag=1
		echo "not found CODIS_CONFIG_PROG"
	fi
	if [ -z "$CODIS_CONFIG_FILE" ] ; then
		errorflag=1
		echo "not found CODIS_CONFIG_FILE"
	fi
	if [ -z "$SERVER_GROUP" ] ; then
		errorflag=1
		echo "not found SERVER_GROUP"
	fi
	if [ -z "$REDIS_CLI" ] ; then
		errorflag=1
		echo "not found REDIS_CLI"
	fi

	if [ $errorflag -eq 1 ] ; then
		exit 0
	fi
}

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
	server_json=/tmp/servergroup.json.$$
	$CODIS_CONFIG_PROG -c $CODIS_CONFIG_FILE server list > $server_json
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

	done <$server_json
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
		sh_log "set $slave_addr:$slave_port slaveof no one,RET:$ret"
		ret=`$REDIS_CLI -h $slave_addr -p $slave_port slaveof $master_addr $master_port`
		sh_log "set $slave_addr:$slave_port slaveof $master_addr $master_port,RET:$ret"
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

SCRIPT_NAME=$0
function main()
{
	sh_log "script [$SCRIPT_NAME] start..."
	load_config

	while [ 1 ]
	do
		get_groups
		check_slave
		sleep 1200
	done
}

if [ $# -ne 1 ] ; then
	echo "usage: $SCRIPT_NAME config_file"
	exit 0
fi

CONFIG_FILE=$1

main

