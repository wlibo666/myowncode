#!/bin/bash

RDBPATH=/letv/run/codis/server
RDBFILE=""
RDBBACK=""
RDBFILESERVER=root@10.148.10.68:/root/rdbfiles/
ETHNAME=eth0
ETHIP=`ifconfig $ETHNAME | grep "inet addr:" | awk '{print $2}' | awk -F: '{print $2}'`

function backup()
{
	while [ 1 ]
	do
		for PORT in "6381" "6382" "6383" "6384" "6385" "6386" "6387"
		do
			RDBFILE=$RDBPATH/codis-server-$PORT.rdb
			RDBBACK=$RDBPATH/$ETHIP-codis-server-$PORT.rdb.latest
			# rename rdb file
			#if [ -f $RDBFILE ] ; then
				echo "sudo /bin/rm $RDBBACK 1>/dev/null 2>/dev/null"
				echo "sudo /bin/mv $RDBFILE $RDBBACK"
				# copy rdbfile
				echo "scp $RDBBACK $RDBFILESERVER"
			#fi
		done
		sleep 300
	done
}

function main()
{
	backup
}

main

