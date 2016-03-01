#!/bin/bash
CODIS_CONFIG_PROG="/letv/codis/bin/codis-config"
CODIS_CONFIG_FILE="/letv/codis/conf/config.ini"
SERVER_GROUP="/tmp/server.group.checkmem"
REDIS_CLI="/letv/codis/bin/redis-cli"

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
	server_json=/tmp/servergroup.json
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
get_groups
