package main

import (
	"fmt"
	"github.com/skynetservices/skynet"
	"github.com/skynetservices/skynet/client"
)

type TutorialRequest struct {
	Value int
}

type TutorialResponse struct {
	Value int
}

func main() {
	config, _ := skynet.GetClientConfig()
	client := client.NewClient(config)

	service := client.GetService("TutorialService", "1", "Development", "")

	req := &TutorialRequest{
		Value: 1,
	}

	resp := &TutorialResponse{}

	err := service.Send(nil, "AddOne", req, resp)

	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(resp.Value)
	}
}
