package main

import (
	"fmt"
	// "os"
)

func main() {

	//if len(os.Args) < 2 {
	//	fmt.Println("Usage: ./mbwebhook [ip address]")
	//} else {

	a := App{}
	a.Initialize()

	a.Run(":" + a.conf.Port)
	fmt.Println("SMS Gateway Initialized on port: ", a.conf.Port)
	//}
}
