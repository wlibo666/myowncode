package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

func store_date(date string) {
	c := fmt.Sprintf("/bin/echo %s > .date", date)
	cmd := exec.Command(c)
	err := cmd.Run()
	if err != nil {
		fmt.Printf("exec [%s] failed,err:%s\n", c, err.Error())
	}
}

func write_date(date string) {
	f, err := os.OpenFile(".date", os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(date)
}

func read_date() string {
	f, err := os.Open(".date")
	if err != nil {
		return ""
	}
	defer f.Close()
	var b []byte = make([]byte, 8)
	f.Read(b)
	return string(b)
}

func main() {
	fmt.Printf("mic [%d]\n", time.Now().UnixNano()/int64(time.Microsecond))
	store_date("20160104")

	write_date("20160105")
	s := read_date()
	fmt.Printf("read data:%s\n", s)
}
