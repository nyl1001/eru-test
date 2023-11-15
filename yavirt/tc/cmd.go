package main

import (
	"fmt"
	"os"
	"os/exec"
)

func executeCommand(command string) error {
	cmd := exec.Command("bash", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func main() {
	// 1. 创建子分类
	classes := []struct {
		parent  string
		classID string
		rate    string
		ceil    string
		burst   string
	}{
		{"1:1", "1:10", "20Mbit", "10Mbit", "15k"},
		{"1:1", "1:20", "3Gbit", "3Gbit", "15k"},
	}

	for _, class := range classes {
		command := fmt.Sprintf("tc class add dev cali5a8d02067ec parent %s classid %s htb rate %s ceil %s burst %s",
			class.parent, class.classID, class.rate, class.ceil, class.burst)
		err := executeCommand(command)
		if err != nil {
			fmt.Println("Error executing command:", err)
			return
		}
	}

	// 2. 避免一个IP霸占带宽资源
	qdiscs := []string{"10", "20"}
	for _, qdisc := range qdiscs {
		command := fmt.Sprintf("tc qdisc add dev cali5a8d02067ec parent 1:%s handle %s: sfq perturb 10", qdisc, qdisc)
		err := executeCommand(command)
		if err != nil {
			fmt.Println("Error executing command:", err)
			return
		}
	}

	// 3. 创建过滤器
	filters := []struct {
		parent  string
		prio    int
		ipRange string
		flowID  string
	}{
		{"1:0", 2, "0.0.0.0/0", "10"},
		{"1:0", 0, "10.0.0.0/8", "20"},
	}

	for _, filter := range filters {
		command := fmt.Sprintf("tc filter add dev cali5a8d02067ec protocol ip parent %s prio %d u32 match ip %s flowid 1:%s",
			filter.parent, filter.prio, filter.ipRange, filter.flowID)
		err := executeCommand(command)
		if err != nil {
			fmt.Println("Error executing command:", err)
			return
		}
	}
}
