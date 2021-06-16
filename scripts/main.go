package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os/exec"
)

type Instance struct {
	InstanceId, PublicIpAddress, InstanceType, Region string
}

func main() {
	MakeRequest()
}

func MakeRequest() {

	// logResult := []string{}

	instances := []Instance{}
	byteValue, _ := ioutil.ReadFile("instances.json")

	json.Unmarshal(byteValue, &instances)

	fmt.Println(instances)
	for index := 0; index < len(instances); index++ {

		fmt.Println("ssh", "-i", "dht-instance.pem", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", "ubuntu@"+instances[index].PublicIpAddress, "bash go/src/github.com/unity-go/scripts/installMongo.sh")

		// fmt.Println("http://" + ips[index] + ":6000/logs")

		cmd, _ := exec.Command("ssh", "-i", "dht-instance.pem", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", "ubuntu@"+instances[index].PublicIpAddress, "free -m").Output()
		fmt.Println(string(cmd), "cmd")

		byteValue, _ := ioutil.ReadFile("logs.txt")
		byteValue = append(byteValue, cmd...)

		data1 := ""
		data1 = string(byteValue) + instances[index].PublicIpAddress + "\n"
		ioutil.WriteFile("logs.txt", []byte(data1), 0644)

	}

}
