package main

import (
	"fmt"
	"github.com/skynetservices/skynet"
	"github.com/skynetservices/skynet/client"
	"github.com/skynetservices/skynet/examples/testing/fibonacci"
	"strconv"
)

func main() {
	config, args := skynet.GetClientConfig()
	client := client.NewClient(config)

	service := client.GetService("Fibonacci", "", "", "")

	if len(args) == 0 {
		fmt.Printf("Usage: %s <positive number>*\n", args[0])
		return
	}

	for _, arg := range args[1:] {
		index, err := strconv.Atoi(arg)
		if err != nil {
			panic(err)
		}
		req := fibonacci.Request{
			Index: index,
		}
		resp := fibonacci.Response{}
		err = service.Send(nil, "Index", req, &resp)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Printf("%d -> %d\n", index, resp.Value)
		}
	}
}
