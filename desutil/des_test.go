package desutil

import (
	"fmt"
	"testing"
)

func TestHello(t *testing.T) {
	fmt.Println("TestHello")
	t.Log("one test passed..") //记录一些你期望记录的信息
}

func Test_Division_1(t *testing.T) {
	t.Error("Division did not work as expected.")
}

func TestWorld(t *testing.T) {
	fmt.Println("TestWorld")

}
