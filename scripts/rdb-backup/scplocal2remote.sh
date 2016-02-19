#!/bin/bash
EXPECTPATH=/usr/bin/expect

if [ $# -ne 4 ] ; then
	echo "$0 user@ipaddr password srcfile dstfile"
	exit 0
fi

REMOTE=$1
USERPWD="$2\r"
SRCFILE=$3
DSTPATH=$4

$EXPECTPATH << EOF
set timeout 30
spawn scp $SRCFILE $REMOTE:$DSTPATH
expect {
	"continue connecting" { send "yes\r"; exp_continue}
	"assword:" { send "$USERPWD\r"}
}
expect eof
EOF




