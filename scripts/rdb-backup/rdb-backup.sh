#!/bin/bash

RDBPATH=/letv/run/codis/server
RDBFILE=""
RDBBACK=""

function backup()
{
	while [ 1 ]
	do
		for PORT in "6381" "6382" "6383" "6384" "6385" "6386" "6387"
		do
			RDBFILE=$RDBPATH/codis-server-$PORT.rdb
			RDBBACK=$RDBFILE."date"

			if [ -f $RDBFILE ] ; then
				sudo /bin/rm $RDBBACK.* 1>/dev/null 2>/dev/null
				sudo /bin/mv $RDBFILE $RDBBACK.`date +%s`
			fi
		done
		sleep 300
	done
}

function main()
{
	backup
}

main

