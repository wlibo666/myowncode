#!/bin/bash
EXPECTPATH=/usr/bin/expect

if [ $# -ne 4 ] ; then
	echo "$0 user@ipaddr password srcfile dstpath"
	exit 0
fi

REMOTE=$1
USERPWD="$2"
SRCFILE=$3
DSTPATH=$4

$EXPECTPATH << EOF
set timeout 1800
spawn scp $REMOTE:$SRCFILE $DSTPATH
expect {
	"continue connecting" { send "yes\r"; exp_continue}
	"assword:" { send "$USERPWD\r"}
}
set timeout 300
send "exit\r"
expect eof
EOF




