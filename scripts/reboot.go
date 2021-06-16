package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os/exec"
	"time"
)

type Instance struct {
	InstanceId, PublicIpAddress, InstanceType, Region string
}

func main() {
	MakeRequest()
}

func MakeRequest() {

	instances := []Instance{}
	byteValue, _ := ioutil.ReadFile("instances.json")

	json.Unmarshal(byteValue, &instances)

	for index := 0; index < len(instances); index++ {
		if index == 0 {
			cmd, _ := exec.Command("aws", "configure", "set", "region", instances[index].Region).Output()
			fmt.Println(string(cmd), "cmd", instances[index].Region)
		} else {
			if instances[index-1].Region != instances[index].Region {
				cmd, _ := exec.Command("aws", "configure", "set", "region", instances[index].Region).Output()
				fmt.Println(string(cmd), "cmd", instances[index].Region)
			}
		}

		cmd, _ := exec.Command("aws", "ec2", "stop-instances", "--instance-ids", instances[index].InstanceId).Output()
		fmt.Println(string(cmd), "cmd", index)

	}

	time.Sleep(10 * time.Minute)

}
