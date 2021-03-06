package main

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"
)

var (
	server1  string = "10.135.29.168:6379"
	server2  string = "10.135.29.171:6379"
	password string = ""
)

var pool1 *redis.Pool
var pool2 *redis.Pool
var SuccTimes int = 0
var FailTimes int = 0

var prefix string = `
[root@localhost conf]# 2015/12/01 17:51:42 read tcp 10.58.65.240:52433->10.58.65.64:2181: i/o timeout
[22599] 01 Dec 17:51:45.377 # slotsmgrt: timeout target 10.58.65.252:6380, lasttime = 1448963489, now = 1448963505
[22599] 01 Dec 17:51:48.995 # slotsmgrt: connect to target 10.58.65.252:6380
[root@localhost conf]# 2015/12/01 17:55:37 read tcp 10.58.65.240:52435->10.58.65.64:2181: i/o timeout
2015/12/01 17:57:50 read tcp 10.58.65.240:52437->10.58.65.64:2181: i/o timeout
[22599] 01 Dec 17:57:56.038 # slotsmgrt: timeout target 10.58.65.252:6380, lasttime = 1448963860, now = 1448963876
[22599] 01 Dec 17:57:57.393 # slotsmgrt: connect to target 10.58.65.252:6380
2015/12/01 17:59:40 read tcp 10.58.65.240:52438->10.58.65.64:2181: i/o timeout
[22599] 01 Dec 18:05:06.565 # slotsmgrt: timeout target 10.58.65.252:6380, lasttime = 1448964290, now = 1448964306
`

func setKeyValue(i int, c redis.Conn) {
	t := strconv.Itoa(i)
	value := "value-" + t + prefix

	/*reply, err := c.Do("GET", "key-"+t)
	if err != nil {
		fmt.Print("GET key-" + t + " failed,err:." + err.Error() + " \n")
		return
	}*/
	//if reply == nil {
	_, err := c.Do("SET", "key-"+t, value)
	if err != nil {
		FailTimes++
	} else {
		SuccTimes++
	}
	//} else {
	//	SuccTimes++
	//}

	//time.Sleep(time.Second)
}

func printResult() {
	fmt.Printf("suc time %d,fail time %d\n", SuccTimes, FailTimes)
}

func poolInit() *redis.Pool {
	return &redis.Pool{
		MaxIdle:     200,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server1)
			if err != nil {
				return nil, err
			}
			//if _, err := c.Do("AUTH", password); err != nil {
			//	c.Close();
			//	return nil, err
			//}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

func pool1Init() *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server2)
			if err != nil {
				return nil, err
			}
			//if _, err := c.Do("AUTH", password); err != nil {
			//	c.Close();
			//	return nil, err
			//}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

// 往redis数据库插入数据
func insertData() {
	var wg sync.WaitGroup
	var begin int = 0
	var setup int = 150000
	pool1 = poolInit()

	for pnum := 0; pnum < 200; pnum++ {
		fmt.Printf("punum[%d],begin[%d],end[%d]\n", pnum, begin, begin+setup)
		go func() {
			wg.Add(1)
			c := pool1.Get()
			for i := begin; i < begin+setup; i++ {
				setKeyValue(i, c)
				if i%5000 == 0 {
					fmt.Printf("now [%s] insert suc %d times, fail %d times.\n", time.Now().String(), SuccTimes, FailTimes)
				}
			}
			defer c.Close()
			wg.Done()
		}()
		begin = begin + setup
	}

	//wg.Done()
	wg.Wait()
}

func manyConn() {
	for i := 0; i < 100000; i++ {
		pool1 = poolInit()
		c := pool1.Get()
		setKeyValue(i, c)
		defer c.Close()
		defer pool1.Close()
		if i%5000 == 0 {
			fmt.Printf("now [%s] suc %d times,fail %d times\n", time.Now().String(), SuccTimes, FailTimes)
		}
	}
}

// 测试迁移时集群是否可用
func clusIsService() {
	pool1 = poolInit()
	pool2 = pool1Init()
	c1 := pool1.Get()
	c2 := pool2.Get()
	for i := 0; i < 300000; i++ {
		var c redis.Conn
		if i%2 == 0 {
			c = c1
		} else {
			c = c2
		}

		t := strconv.Itoa(i)
		reply, err := c.Do("GET", "key-"+t)
		if err == nil {
			SuccTimes++
			if reply == nil {
				fmt.Printf("fail GET key-%s,not found value\n", t)
			} else {
				//fmt.Printf("suc GET key-%s,value[%s]\n", t, reply)
			}
		} else {
			FailTimes++
			fmt.Printf("GET key-%s failed,err %s\n", t, err.Error())
		}
		var st time.Duration
		st = time.Second * 30
		//time.Sleep(st)

		if SuccTimes%1000 == 0 {
			fmt.Printf("now get key success times %d,fail times %d\n", SuccTimes, FailTimes)
			fmt.Printf("suc GET key-%s,value[%s]\n", t, reply)
		}
		if i%5000 == 0 {
			fmt.Printf("now %s will sleep %v\n", time.Now().String(), st)
			time.Sleep(st)
		}
	}
	defer c1.Close()
	defer c2.Close()
	defer pool1.Close()
	defer pool2.Close()
	fmt.Printf("end get key success times %d,fail times %d\n", SuccTimes, FailTimes)
}

func insertHashData() {
	pool1 = poolInit()
	c := pool1.Get()
	keyname := "hashKey"
	for i := 550000; i < 2000000; i++ {
		t := strconv.Itoa(i)
		field := ("field-" + t)
		value := ("hashvalue-" + t + prefix)

		_, err := c.Do("HSET", keyname, field, value)
		if err != nil {
			FailTimes++
			fmt.Printf("HSET %s %s failed,err:%s.\n", keyname, field, err.Error())
		} else {
			//fmt.Printf("reply is %v\n", reply)
			SuccTimes++
		}

		if i%5000 == 0 {
			fmt.Printf("now [%v] insert data number:%d\n", time.Now(), i)
		}
	}
	c.Close()
	pool1.Close()
}

func usage(program string) {
	fmt.Printf("%s cmdtype redisaddr\n", program)
	fmt.Printf("    cmd{insertData | getData | insertHash | manyConn}  10.98.28.25:6381\n")
}

func main() {
	if len(os.Args) != 3 {
		usage(os.Args[0])
		os.Exit(0)
	}
	server1 = os.Args[2]
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		fmt.Printf("succTimes:%d,failTimes:%d\n", SuccTimes, FailTimes)
		os.Exit(0)
	}()
	switch os.Args[1] {
	case "insertData":
		insertData()
	case "getData":
		clusIsService()
	case "insertHash":
		insertHashData()
	case "manyConn":
		manyConn()
	default:
		usage(os.Args[0])
		os.Exit(0)
	}
	printResult()

}
