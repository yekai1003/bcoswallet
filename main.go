//并非在GOPATH目录 -- go mod
package main

import (
	"bcoswallet/cmd"
	"fmt"
	"os"
)

func main() {
	url := os.Getenv("ETH_URL")
	if url == "" {
		url = "http://127.0.0.1:8545"
	}
	fmt.Println("url:", url)
	cli := cmd.NewCLI("./data/", url, "tokens.json")
	cli.Run()
}
