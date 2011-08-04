//Copyright (c) 2011 Brian Ketelsen

//Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

//The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

//THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package skylib

import (
	"log"
	"json"
	"flag"
	"os"
	"fmt"
	"rand"
	"rpc"
	"rpc/jsonrpc"
	"expvar"
	"strings"
)


var NS *NetworkServers
//var RpcServices []*RpcServer


var Port *int = flag.Int("port", 9999, "tcp port to listen")
var Name *string = flag.String("name", os.Args[0], "name of this Agent")
var BindIP *string = flag.String("bindaddress", "127.0.0.1", "address to bind")
var LogFileName *string = flag.String("logFileName", "myservice.log", "name of logfile")
var LogLevel *int = flag.Int("logLevel", 5, "log level (1-5)")
var Protocol *string = flag.String("protocol", "http+gob", "RPC message transport protocol (default is http+gob; try json")
var Requests *expvar.Int
var Errors *expvar.Int
var Goroutines *expvar.Int
//var svc *Service


func GetServiceProviders(provides string) (providesList []*RpcServer) {
	for _, v := range NS.Services {
		if v != nil && v.Provides == provides {
			providesList = append(providesList, v)
		}
	}
	return
}

// This is simple today - it returns the first listed service that matches the request
// Load balancing needs to be applied here somewhere.
func GetRandomClientBySignature(provides string) (*rpc.Client, os.Error) {
	var newClient *rpc.Client
	var err os.Error
	serviceList := GetServiceProviders(provides)

	if len(serviceList) > 0 {
		chosen := rand.Int() % len(serviceList)
		s := serviceList[chosen]

		hostString := fmt.Sprintf("%s:%d", s.IPAddress, s.Port)
		protocol := strings.ToLower(s.Protocol) // to be safe
		switch protocol {
		default:
			newClient, err = rpc.DialHTTP("tcp", hostString)
		case "json":
			newClient, err = jsonrpc.Dial("tcp", hostString)
		}

		if err != nil {
			LogWarn(fmt.Sprintf("Found %d nodes to provide service %s requested on %s, but failed to connect.",
				len(serviceList), provides, hostString))
			return nil, NewError(NO_CLIENT_PROVIDES_SERVICE, provides)
		}

	} else {
		LogWarn(fmt.Sprintf("Found no node to provide service %s.", provides))
		return nil, NewError(NO_CLIENT_PROVIDES_SERVICE, provides)
	}
	return newClient, nil
}


// on startup load the configuration file. 
// After the config file is loaded, we set the global config file variable to the
// unmarshaled data, making it useable for all other processes in this app.
func LoadConfig() {
	data, _, err := DC.Get("/servers/config/networkservers.conf", nil)
	if err != nil {
		log.Panic(err.String())
	}
	if len(data) > 0 {
		setConfig(data)
		return
	}
	LogError("Error loading default config - no data found")
	NS = &NetworkServers{}
}

func RemoveServiceAt(i int) {

	newServices := make([]*RpcServer, 0)

	for k, v := range NS.Services {
		if k != i {
			if v != nil {
				newServices = append(newServices, v)
			}
		}
	}
	NS.Services = newServices
	b, err := json.Marshal(NS)
	if err != nil {
		log.Panic(err.String())
	}
	rev, err := DC.Rev()
	if err != nil {
		log.Panic(err.String())
	}
	_, err = DC.Set("/servers/config/networkservers.conf", rev, b)
	if err != nil {
		log.Panic(err.String())
	}

}

func RemoveFromConfig(r *RpcServer) {

	newServices := make([]*RpcServer, 0)

	for _, v := range NS.Services {
		if v != nil {
			if !v.Equal(r) {
				newServices = append(newServices, v)
			}

		}
	}
	NS.Services = newServices
	b, err := json.Marshal(NS)
	if err != nil {
		log.Panic(err.String())
	}
	rev, err := DC.Rev()
	if err != nil {
		log.Panic(err.String())
	}
	_, err = DC.Set("/servers/config/networkservers.conf", rev, b)
	if err != nil {
		log.Panic(err.String())
	}
}

func AddToConfig(r *RpcServer) {
	for _, v := range NS.Services {
		if v != nil {
			if v.Equal(r) {
				LogInfo(fmt.Sprintf("Skipping adding %s : alreday exists.", v.Provides))
				return // it's there so we don't need an update
			}
		}
	}
	NS.Services = append(NS.Services, r)
	LogDebug("Added", r.Provides, r.Protocol)
	b, err := json.Marshal(NS)
	if err != nil {
		log.Panic(err.String())
	}
	rev, err := DC.Rev()
	if err != nil {
		log.Panic(err.String())
	}
	_, err = DC.Set("/servers/config/networkservers.conf", rev, b)
	if err != nil {
		log.Panic(err.String())
	}
}

// unmarshal data from remote store into global config variable
func setConfig(data []byte) {
	err := json.Unmarshal(data, &NS)
	if err != nil {
		log.Panic(err.String())
	}
}

// Watch for remote changes to the config file.  When new changes occur
// reload our copy of the config file.
// Meant to be run as a goroutine continuously.
func WatchConfig() {
	rev, err := DC.Rev()
	if err != nil {
		log.Panic(err.String())
	}

	for {
		// blocking wait call returns on a change
		ev, err := DC.Wait("/servers/config/networkservers.conf", rev)
		if err != nil {
			log.Panic("Error waiting on config: " + err.String())
		}
		log.Println("Received new configuration.  Setting local config.")
		setConfig(ev.Body)

		rev = ev.Rev + 1
	}

}
