#!/bin/bash
REDIS_CLI=/letv/codis/bin/redis-cli
DEV_LIST=/tmp/alldevice.list
DATE=$(date +%Y-%m-%d)
PHONE_NUM=0
TV_NUM=0
ROUTER_NUM=0
SERVER_GROUP=/tmp/server.group

function GetDevFromSlave()
{
	rm $DEV_LIST 1>/dev/null 2>/dev/null
	while read line
	do
		flag=`echo "$line" | grep "slave"`
		if [ "$flag" != "" ] ; then
			slave=`echo "$line" | awk -F' ' '{print $1}'`
			slave_addr=`echo "$slave" | awk -F: '{print $1}'`
			slave_port=`echo "$slave" | awk -F: '{print $2}'`
			$REDIS_CLI -h $slave_addr -p $slave_port keys 'db_rapp_*' >> $DEV_LIST
		fi
	done <$SERVER_GROUP
}

function StatisDevNum()
{
	PHONE_NUM=`cat $DEV_LIST | grep "db_rapp_--" | wc -l`
	ROUTER_NUM=`cat $DEV_LIST | grep "db_rapp_LetvWiFi" | wc -l`
	ALL_NUM=`cat $DEV_LIST | wc -l`
	TV_NUM=`expr $ALL_NUM - $PHONE_NUM - $ROUTER_NUM`

	return	
}

function PrintDevNum()
{
	echo -e "moblie $PHONE_NUM\nrouter $ROUTER_NUM\ntv $TV_NUM" > devnum.log$DATE
}

function StatisDev()
{
	GetDevFromSlave
	StatisDevNum
	PrintDevNum
}
`pwd`/getServer.sh
StatisDev
find /letv/codis/cron -name "*.log*" -mtime +2 -exec rm {} \;
