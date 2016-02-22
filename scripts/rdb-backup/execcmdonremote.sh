#!/bin/bash
EXPECTPATH=/usr/bin/expect

if [ $# -ne 3 ] ; then
	echo "$0 user@ipaddr password cmdline"
	exit 1
fi

REMOTE=$1
USERPWD="$2\r"
CMD=$3

$EXPECTPATH << EOF
set timeout 1800
spawn ssh $REMOTE $CMD
expect {
	"continue connecting" { send "yes\r"; exp_continue}
	"assword:" { send "$USERPWD\r"}
}
expect eof
EOF



