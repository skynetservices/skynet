//Copyright (c) 2012 Brian Ketelsen

//Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

//The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

//THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
package main

import (
	"github.com/skynetservices/skynet"
	"github.com/skynetservices/skynet/service"
	"log"
	"os"
	"strings"
)

type TestService struct {
	Log skynet.SemanticLogger
}

func (s *TestService) Registered(service *service.Service)   {}
func (s *TestService) Unregistered(service *service.Service) {}
func (s *TestService) Started(service *service.Service)      {}
func (s *TestService) Stopped(service *service.Service) {
	s.Log.Trace("Stopped")
}

func NewTestService() *TestService {
	r := &TestService{}
	return r
}

func (s *TestService) Upcase(requestInfo *skynet.RequestInfo, in map[string]interface{}, out map[string]interface{}) (err error) {
	out["data"] = strings.ToUpper(in["data"].(string))
	return
}

func main() {
	testService := NewTestService()

	config, _ := skynet.GetServiceConfig()

	if config.Name == "" {
		config.Name = "TestService"
	}

	if config.Version == "" {
		config.Version = "1"
	}

	if config.Region == "unknown" {
		config.Region = "Clearwater"
	}

	var err error
	mlogger, err := skynet.NewMongoSemanticLogger("localhost", "skynet",
		"log", config.UUID)
	clogger := skynet.NewConsoleSemanticLogger("TestService", os.Stdout)
	testService.Log = skynet.NewMultiSemanticLogger(mlogger, clogger)
	config.Log = testService.Log
	if err != nil {
		config.Log.Trace("Could not connect to mongo db for logging")
	}
	service := service.CreateService(testService, config)

	// handle panic so that we remove ourselves from the pool in case
	// of catastrophic failure
	defer func() {
		service.Shutdown()
		if err := recover(); err != nil {
			log.Println("Unrecovered error occured: ", err)
		}
	}()

	// If we pass false here service will not be Registered we could
	// do other work/tasks by implementing the Started method and
	// calling Register() when we're ready
	waiter := service.Start(true)

	// waiting on the sync.WaitGroup returned by service.Start() will
	// wait for the service to finish running.
	waiter.Wait()
}
