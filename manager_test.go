package Executor

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	fmt.Println("超时测试")
	n := NewManager("OvO")
	ctx, _ := context.WithTimeout(context.Background(), 4*time.Second)
	pid, err := n.NewProcessWithContext(
		ctx,
		"bash",
		[]string{"-c", "/bin/bash"},
		"",
	)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("等待结果")
	stdout, stderr, err := n.WaitOutput(pid)
	fmt.Println("获得结果")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Stdout:", string(stdout))
	fmt.Println("Stderr:", string(stderr))
}
