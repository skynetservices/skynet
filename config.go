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
	"sync"
	"time"
)

var portMutex sync.Mutex

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
		ba.IPAddress = "0.0.0.0"
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

func (ba *BindAddr) String() string {
	if ba == nil {
		return ""
	}
	return fmt.Sprintf("%s:%d", ba.IPAddress, ba.Port)
}

func (ba *BindAddr) Listen() (listener *net.TCPListener, err error) {
	// Ensure Admin, and RPC don't fight over the same port
	portMutex.Lock()
	defer portMutex.Unlock()

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
	Log                  Logger `json:"-"`
	UUID                 string
	Name                 string
	Version              string
	Region               string
	ServiceAddr          *BindAddr
	AdminAddr            *BindAddr
	DoozerConfig         *DoozerConfig `json:"-"`
	DoozerUpdateInterval time.Duration `json:"-"`
}

type ServiceStatistics struct {
	Clients        int32
	StartTime      string
	LastRequest    string
	RequestsServed int64

	// For now this will be since startup, we might change it later to be for a given sample interval
	AverageResponseTime time.Duration
	TotalDuration       time.Duration `json:"-"`
}

type ServiceInfo struct {
	Config     *ServiceConfig
	Registered bool
	Stats      ServiceStatistics
}

func (s *ServiceInfo) GetConfigPath() string {
	return "/services/" + s.Config.Name + "/" + s.Config.Version + "/" + s.Config.Region + "/" + s.Config.ServiceAddr.IPAddress + "/" + strconv.Itoa(s.Config.ServiceAddr.Port)
}

type ClientConfig struct {
	Log                       Logger        `json:"-"`
	DoozerConfig              *DoozerConfig `json:"-"`
	IdleConnectionsToInstance int
	MaxConnectionsToInstance  int
	IdleTimeout               time.Duration
}

func GetDefaultEnvVar(name, def string) (v string) {
	v = os.Getenv(name)
	if v == "" {
		v = def
	}
	return
}

func FlagsForDoozer(dcfg *DoozerConfig, flagset *flag.FlagSet) {
	flagset.StringVar(&dcfg.Uri, "doozer", GetDefaultEnvVar("SKYNET_DZHOST", DefaultDoozerdAddr), "initial doozer instance to connect to")
	flagset.StringVar(&dcfg.BootUri, "doozerboot", GetDefaultEnvVar("SKYNET_DZNSHOST", DefaultDoozerdAddr), "initial doozer instance to connect to")
	flagset.BoolVar(&dcfg.AutoDiscover, "autodiscover", GetDefaultEnvVar("SKYNET_DZDISCOVER", "true") == "true", "auto discover new doozer instances")
}

func FlagsForClient(ccfg *ClientConfig, flagset *flag.FlagSet) {
	FlagsForDoozer(ccfg.DoozerConfig, flagset)
	flagset.DurationVar(&ccfg.IdleTimeout, "timeout", DefaultIdleTimeout, "amount of idle time before timeout")
	flagset.IntVar(&ccfg.IdleConnectionsToInstance, "maxidle", DefaultIdleConnectionsToInstance, "maximum number of idle connections to a particular instance")
	flagset.IntVar(&ccfg.MaxConnectionsToInstance, "maxconns", DefaultMaxConnectionsToInstance, "maximum number of concurrent connections to a particular instance")

}

func GetClientConfig() (config *ClientConfig, args []string) {
	return GetClientConfigFromFlags(os.Args[1:])
}

func GetClientConfigFromFlags(argv []string) (config *ClientConfig, args []string) {
	config = &ClientConfig{
		DoozerConfig: &DoozerConfig{},
	}

	flagset := flag.NewFlagSet("config", flag.ContinueOnError)

	FlagsForClient(config, flagset)

	err := flagset.Parse(argv)

	args = flagset.Args()
	if err == flag.ErrHelp {
		// -help was given, pass it on to caller who 
		// may decide to quit instead of continuing
		args = append(args, "-help")
	}

	return
}

func FlagsForService(scfg *ServiceConfig, flagset *flag.FlagSet) {
	FlagsForDoozer(scfg.DoozerConfig, flagset)
	flagset.StringVar(&scfg.UUID, "uuid", UUID(), "UUID for this service")
	flagset.StringVar(&scfg.Region, "region", GetDefaultEnvVar("SKYNET_REGION", DefaultRegion), "region service is located in")
	flagset.StringVar(&scfg.Version, "version", DefaultVersion, "version of service")
	flagset.DurationVar(&scfg.DoozerUpdateInterval, "dzupdate", DefaultDoozerUpdateInterval, "ns to wait before sending the next status update")
}

func GetServiceConfig() (config *ServiceConfig, args []string) {
	return GetServiceConfigFromFlags(os.Args[1:])
}

func GetServiceConfigFromFlags(argv []string) (config *ServiceConfig, args []string) {

	config = &ServiceConfig{
		DoozerConfig: &DoozerConfig{},
	}

	flagset := flag.NewFlagSet("config", flag.ContinueOnError)

	FlagsForService(config, flagset)

	rpcAddr := flagset.String("l", GetDefaultBindAddr(), "host:port to listen on for RPC")
	adminAddr := flagset.String("admin", GetDefaultBindAddr(), "host:port to listen on for admin")

	err := flagset.Parse(argv)
	args = flagset.Args()
	if err == flag.ErrHelp {
		// -help was given, pass it on to caller who 
		// may decide to quit instead of continuing
		args = append(args, "-help")
	}

	rpcBA, err := BindAddrFromString(*rpcAddr)
	if err != nil {
		panic(err)
	}
	adminBA, err := BindAddrFromString(*adminAddr)
	if err != nil {
		panic(err)
	}

	config.ServiceAddr = rpcBA
	config.AdminAddr = adminBA

	return
}
