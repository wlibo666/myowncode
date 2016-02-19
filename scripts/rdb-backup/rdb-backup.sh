#!/bin/bash

CODIS_CONFIG_PROG=`pwd`/codis-config
CODIS_CONFIG_FILE=`pwd`/config.ini
SCP_CMD=`pwd`/scpremote2local.sh

SRCRDBPATH=/letv/run/codis/server
DSTRDBPATH=/letv/letv-redis/rdbbackup
USERNAME=root
USERPWD="fW+ug3dz1w9XGYaCApy5"

SERVER_GROUP_TEM=/tmp/server.group
SERVER_GROUP_ALL=$SERVER_GROUP_TEM.*
SERVER_GROUP=$SERVER_GROUP_TEM.$$

SCRIPT_NAME=$0
LOG_FILE=`pwd`/rdb-backup.log

function sh_log()
{
	echo "`date`: $1 $2 $3 $4" >> $LOG_FILE
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
	SERVER_JSON=/tmp/servergroup.json.$$
	rm -rf /tmp/servergroup.json.* 1>/dev/null 2>/dev/null
	rm -f $SERVER_GROUP_ALL 1>/dev/null 2>/dev/null
	$CODIS_CONFIG_PROG -c $CODIS_CONFIG_FILE server list > $SERVER_JSON
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
	done <$SERVER_JSON
}

function rdb_backup()
{
	ipaddr=`echo "$1" | awk -F: '{print $1}'`
	port=`echo "$1" | awk -F: '{print $2}'`

	rdbfilename=$SRCRDBPATH/codis-server-$port.rdb
	tmpdstfilename=$DSTRDBPATH/$ipaddr-codis-server-$port.rdb.latest
	dstfilename=$DSTRDBPATH/$ipaddr-codis-server-$port.rdb
	
	remote=$USERNAME@$ipaddr
	$SCP_CMD $remote $USERPWD $rdbfilename $tmpdstfilename
	if [ -f $tmpdstfilename ] ; then
		mv $tmpdstfilename $dstfilename
		sh_log "rdb_backup file $dstfilename success."
	fi
}

function slave_rdb_backup()
{
	while read line
	do
		flag=`echo "$line" | grep "slave"`
		if [ "$flag" != "" ] ; then
			slave=`echo "$line" | awk -F' ' '{print $1}'`
			if [ "$slave" != "" ] ; then
				sh_log "begin rdb_backup $slave"
				rdb_backup "$slave"
			fi
		fi
	done <$SERVER_GROUP
}


function main()
{
	sh_log "$SCRIPT_NAME start..."
	mkdir -p $DSTRDBPATH
	while [ 1 ]
	do
		get_groups
		slave_rdb_backup
		sleep 7200
	done
}

main
