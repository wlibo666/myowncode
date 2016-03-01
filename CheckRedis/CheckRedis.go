package main

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"os"
	"strings"
	"time"
)

func InitRedisPool(redisAddr string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", redisAddr, redis.DialConnectTimeout(10*time.Second))
			if err != nil {
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

func CheckRedis(redisAddr string) int64 {
	pool := InitRedisPool(redisAddr)
	if pool == nil {
		return FAIL_INIT_REDIS_POOL
	}
	defer pool.Close()
	c := pool.Get()
	if c == nil {
		return FAIL_GET_REDIS_CONN
	}
	defer c.Close()

	tStart := time.Now().UnixNano()
	_, err := c.Do("info")
	if err != nil {
		e := err.Error()
		if strings.Contains(e, "missing port") {
			return FAIL_MISS_PORT
		}
		if strings.Contains(e, "connection refused") {
			return FAIL_SERVER_REFUSE
		}
		if strings.Contains(e, "timed out") || strings.Contains(e, "timeout") {
			return FAIL_TIMEOUT
		}
		return FAIL_SERVER
	}
	tEnd := time.Now().UnixNano()
	return (tEnd - tStart) / 1000
}

func usage(cmdname string) {
	fmt.Printf("check redis's service state.\n ")
	fmt.Printf("Usage: %s redisAddr\n", cmdname)
	fmt.Printf("  it return below number:\n")
	fmt.Printf("    1 : init redis pool failed\n")
	fmt.Printf("    2 : get redis connection from pool failed\n")
	fmt.Printf("    3 : miss redis's port\n")
	fmt.Printf("    4 : redis refused\n")
	fmt.Printf("    5 : redis server error\n")
	fmt.Printf("    6 : connect time out\n")
	fmt.Printf("    other number: used time of execing INFO(microsecond)\n")

	os.Exit(2)
}

var (
	FAIL_INIT_REDIS_POOL = int64(1)
	FAIL_GET_REDIS_CONN  = int64(2)
	FAIL_MISS_PORT       = int64(3)
	FAIL_SERVER_REFUSE   = int64(4)
	FAIL_SERVER          = int64(5)
	FAIL_TIMEOUT         = int64(6)
)

var redisAddr string

func main() {
	if len(os.Args) != 2 {
		usage(os.Args[0])
	}
	redisAddr = os.Args[1]
	res := CheckRedis(redisAddr)
	fmt.Printf("%d", res)
}
