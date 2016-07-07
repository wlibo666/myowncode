package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/wandoulabs/zkhelper"
	"sort"
)

type MigrateTaskInfo struct {
	SlotId     int    `json:"slot_id"`
	NewGroupId int    `json:"new_group"`
	Delay      int    `json:"delay"`
	CreateAt   string `json:"create_at"`
	Percent    int    `json:"percent"`
	Status     string `json:"status"`
	Id         string `json:"-"`
}

type MigrateTask struct {
	MigrateTaskInfo
	zkConn      zkhelper.Conn
	productName string
}

var gZkConn zkhelper.Conn

func ZkConnInit(zkAddr string) error {
	var err error
	fmt.Printf("now connect zk[%s].\n", zkAddr)
	gZkConn, err = zkhelper.ConnectToZk(zkAddr, 30000)
	if err != nil {
		return errors.New("connect to zk failed")
	}
	return nil
}

func ZkConnClose() {
	gZkConn.Close()
}

func getMigrateTasksPath(product string) string {
	return fmt.Sprintf("/zk/codis/db_%s/migrate_tasks", product)
}

type AllTasks []MigrateTaskInfo

func (t AllTasks) Len() int {
	return len(t)
}

func (t AllTasks) Less(i, j int) bool {
	return t[i].Id <= t[j].Id
}

func (t AllTasks) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func Tasks(product string) []MigrateTaskInfo {
	res := AllTasks{}
	tasks, _, _ := gZkConn.Children(getMigrateTasksPath(product))
	for _, id := range tasks {
		data, _, _ := gZkConn.Get(getMigrateTasksPath(product) + "/" + id)
		info := new(MigrateTaskInfo)
		json.Unmarshal(data, info)
		info.Id = id
		res = append(res, *info)
	}
	sort.Sort(res)
	return res
}

func DelAllTasks(zk string, product string) error {
	err := ZkConnInit(zk)
	if err != nil {
		return err
	}
	fmt.Printf("zookeeper conn init success.\n")
	tasks := Tasks(product)
	if len(tasks) == 0 {
		return errors.New("task number is 0")
	}
	fmt.Printf("get all migrate task success.\n")
	for _, task := range tasks {
		fmt.Printf("now will delete task id [%s],slot id [%d], newGroupId [%d], status [%s]\n",
			task.Id, task.SlotId, task.NewGroupId, task.Status)
		gZkConn.Delete(getMigrateTasksPath(product)+"/"+task.Id, -1)
	}
	ZkConnClose()
	return nil
}
