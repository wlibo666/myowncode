package main

import (
	"log"
	"os"
)

func main() {
	file, _ := os.Create("testLog.log")
	logger := log.New(file, "", log.LstdFlags)
	defer file.Close()
	logger.Println("hhh")
	logger.Printf("this is formt %s", "format")
}
