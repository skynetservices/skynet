//Copyright (c) 2011 Brian Ketelsen

//Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

//The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

//THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package skynet

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type BindAddr struct {
	IPAddress string
	Port      int
	MaxPort   int
}

func BindAddrFromString(host string) (ba *BindAddr, err error) {
	if host == "" {
		return
	}
	split := strings.Index(host, ":")
	if split == -1 {
		err = errors.New(fmt.Sprintf("Must specify a port for address (got %q)", host))
		return
	}

	ba = &BindAddr{}

	ba.IPAddress = host[:split]
	if ba.IPAddress == "" {
		ba.IPAddress = "127.0.0.1"
	}

	portstr := host[split+1:]
	if ba.Port, err = strconv.Atoi(portstr); err == nil {
		return
	}

	var rindex int
	if rindex = strings.Index(portstr, "-"); rindex == -1 {
		err = errors.New(fmt.Sprintf("Couldn't process port for %q: %v", host, err))
		return
	}

	maxPortStr := portstr[rindex+1:]
	portstr = portstr[:rindex]

	if ba.Port, err = strconv.Atoi(portstr); err != nil {
		err = errors.New(fmt.Sprintf("Couldn't process port for %q: %v", host, err))
		return
	}
	if ba.MaxPort, err = strconv.Atoi(maxPortStr); err != nil {
		err = errors.New(fmt.Sprintf("Couldn't process port for %q: %v", host, err))
		return
	}

	return
}

func (ba BindAddr) String() string {
	return fmt.Sprintf("%s:%d", ba.IPAddress, ba.Port)
}

func (ba *BindAddr) Listen() (listener *net.TCPListener, err error) {
	for {
		var laddr *net.TCPAddr
		laddr, err = net.ResolveTCPAddr("tcp", ba.String())
		if err != nil {
			panic(err)
		}
		listener, err = net.ListenTCP("tcp", laddr)
		if err == nil {
			return
		}
		if ba.Port < ba.MaxPort {
			ba.Port++
		} else {
			return
		}

	}
	return
}

type ServiceConfig struct {
	Log          Logger `json:"-"`
	UUID         string
	Name         string
	Version      string
	Region       string
	ServiceAddr  *BindAddr
	AdminAddr    *BindAddr
	DoozerConfig *DoozerConfig `json:"-"`
}

type ClientConfig struct {
	Log                Logger        `json:"-"`
	DoozerConfig       *DoozerConfig `json:"-"`
	ConnectionPoolSize int
	IdleTimeout        time.Duration
}

func getDefaultEnvVar(name, def string) (v string) {
	v = os.Getenv(name)
	if v == "" {
		v = def
	}
	return
}

func GetServiceConfigFromFlags(argv ...string) (config *ServiceConfig, args []string) {
	flagset := flag.NewFlagSet("config", flag.ContinueOnError)

	var (

		// bindPort       *int    = flagset.Int("port", 9999, "tcp port to listen")
		// adminPort      *int    = flagset.Int("adminport", 9998, "tcp port to listen for admin")
		// bindAddr       *string = flagset.String("address", "127.0.0.1", "address to bind")
		rpcAddr        *string = flagset.String("l", getDefaultEnvVar("SKYNET_LISTEN", ":9999"), "host:port to listen on for RPC")
		adminAddr      *string = flagset.String("admin", getDefaultEnvVar("SKYNET_ADMIN", ":9998"), "host:port to listen on for admin")
		region         *string = flagset.String("region", getDefaultEnvVar("SKYNET_REGION", "unknown"), "region service is located in")
		doozer         *string = flagset.String("doozer", getDefaultEnvVar("DZHOST", "127.0.0.1:8046"), "initial doozer instance to connect to")
		doozerBoot     *string = flagset.String("doozerboot", getDefaultEnvVar("DZNSHOST", "127.0.0.1:8046"), "initial doozer instance to connect to")
		doozerDiscover *bool   = flagset.Bool("autodiscover", getDefaultEnvVar("DZDISCOVER", "true") == "true", "auto discover new doozer instances")
		uuid           *string = flagset.String("uuid", "", "UUID for this service")
	)

	if len(argv) == 0 {
		argv = os.Args[1:]
	}
	flagset.Parse(argv)
	args = flagset.Args()

	if *uuid == "" {
		*uuid = UUID()
	}

	rpcBA, err := BindAddrFromString(*rpcAddr)
	if err != nil {
		panic(err)
	}
	adminBA, err := BindAddrFromString(*adminAddr)
	if err != nil {
		panic(err)
	}

	config = &ServiceConfig{
		UUID:        *uuid,
		Region:      *region,
		ServiceAddr: rpcBA,
		AdminAddr:   adminBA,
		DoozerConfig: &DoozerConfig{
			Uri:          *doozer,
			BootUri:      *doozerBoot,
			AutoDiscover: *doozerDiscover,
		},
	}

	return
}
