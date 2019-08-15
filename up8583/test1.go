package up8583

import (
	"fmt"
	log "github.com/jeanphorn/log4go"
)

func Test() {
	fmt.Println("test aaa...")
	log.LOGGER("Test1").Debug("category Test1 debug test aaaa ...")
}
