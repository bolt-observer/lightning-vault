package main

import (
	"fmt"
	"github.com/bolt-observer/macaroon_vault/utils"
)

func main() {
        // A client library is already included in this project
	ret, err := utils.GetData("lnd1", "")
	if err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", ret)
}
