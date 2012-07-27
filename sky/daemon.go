package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/bketelsen/skynet"
	"github.com/bketelsen/skynet/client"
	"github.com/bketelsen/skynet/service"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

// Daemon() will run and maintain skynet services.
//
// Daemon() will initially deploy those specified in the file given in the "-config" option
//
// Daemon() will run the "SkynetDeployment" service, which can be used to remotely spawn
// new services on the host.
func Daemon(q *client.Query, argv []string) {

	config, args := skynet.GetServiceConfigFromFlags(argv...)

	config.Name = "SkynetDaemon"
	config.Version = "1"
	config.Region = "Jersey"

	var err error
	mlogger, err := skynet.NewMongoLogger("localhost", "skynet", "log", config.UUID)
	clogger := skynet.NewConsoleLogger(os.Stdout)
	config.Log = skynet.NewMultiLogger(mlogger, clogger)
	if err != nil {
		config.Log.Item("Could not connect to mongo db for logging")
	}

	deployment := &SkynetDaemon{
		Log:      config.Log,
		Services: map[string]*SubService{},
	}

	s := service.CreateService(deployment, config)

	// handle panic so that we remove ourselves from the pool in case of catastrophic failure
	defer func() {
		s.Shutdown()
		if err := recover(); err != nil {
			log.Println("Unrecovered error occured: ", err)
		}
	}()

	if len(args) == 1 {
		err := deployConfig(deployment, args[0])
		if err != nil {
			config.Log.Item(err)
		}
	}

	// If we pass false here service will not be Registered
	// we could do other work/tasks by implementing the Started method and calling Register() when we're ready
	s.Start(true).Wait()
}

// deploy each of the services listed in the provided file
func deployConfig(s *SkynetDaemon, cfg string) (err error) {
	cfgFile, err := os.Open(cfg)
	if err != nil {
		return
	}
	br := bufio.NewReader(cfgFile)
	for {
		var bline []byte
		var prefix bool
		bline, prefix, err = br.ReadLine()
		if err == io.EOF {
			err = nil
			break
		}
		if err != nil {
			return
		}
		if prefix {
			err = errors.New("Config line to long in " + cfg)
			return
		}
		line := strings.TrimSpace(string(bline))
		if len(line) == 0 {
			continue
		}

		split := strings.Index(line, " ")
		if split == -1 {
			split = len(line)
		}
		servicePath := line[:split]
		args := strings.TrimSpace(line[split:])
		s.Deploy(&skynet.RequestInfo{}, M{"service": servicePath, "args": args}, &M{})
	}
	return
}

// SkynetDaemon is a service for administering other services
type SkynetDaemon struct {
	Log         skynet.Logger
	Services    map[string]*SubService
	serviceLock sync.Mutex
}

func (sd *SkynetDaemon) Registered(s *service.Service)   {}
func (sd *SkynetDaemon) Unregistered(s *service.Service) {}
func (sd *SkynetDaemon) Started(s *service.Service)      {}
func (sd *SkynetDaemon) Stopped(s *service.Service) {
	sd.StopAllSubServices(&skynet.RequestInfo{}, M{}, &M{})
}

func (s *SkynetDaemon) Deploy(requestInfo *skynet.RequestInfo, in M, out *M) (err error) {
	*out = map[string]interface{}{}
	uuid := skynet.UUID()
	(*out)["uuid"] = uuid

	servicePath := in["service"].(string)
	args := in["args"].(string)

	s.Log.Item(SubserviceDeployment{
		ServicePath: servicePath,
		Args:        args,
	})

	ss, err := NewSubService(s.Log, servicePath, args, uuid)
	if err != nil {
		return
	}
	s.serviceLock.Lock()
	s.Services[uuid] = ss
	s.serviceLock.Unlock()
	return
}

func (s *SkynetDaemon) getSubService(uuid string) (ss *SubService) {
	s.serviceLock.Lock()
	ss = s.Services[uuid]
	s.serviceLock.Unlock()
	return
}

type M map[string]interface{}

func (m M) String(key string) (val string, ok bool) {
	vali, ok := m[key]
	if !ok {
		return
	}
	val, ok = vali.(string)
	return
}

func (s *SkynetDaemon) ListSubServices(requestInfo *skynet.RequestInfo, in M, out M) (err error) {
	out["Services"] = s.Services
	return
}

func (s *SkynetDaemon) StopAllSubServices(requestInfo *skynet.RequestInfo, in M, out *M) (err error) {
	*out = map[string]interface{}{}
	var uuids []string
	s.serviceLock.Lock()
	for uuid := range s.Services {
		uuids = append(uuids, uuid)
	}
	s.serviceLock.Unlock()
	for _, uuid := range uuids {
		err = s.StopSubService(requestInfo, M{"uuid": uuid}, &M{})
		if err != nil {
			return
		}
	}
	return
}

func (s *SkynetDaemon) StartAllSubServices(requestInfo *skynet.RequestInfo, in M, out *M) (err error) {
	*out = map[string]interface{}{}
	var uuids []string
	s.serviceLock.Lock()
	for uuid := range s.Services {
		uuids = append(uuids, uuid)
	}
	s.serviceLock.Unlock()
	for _, uuid := range uuids {
		err = s.StartSubService(requestInfo, M{"uuid": uuid}, &M{})
		if err != nil {
			return
		}
	}
	return
}

func (s *SkynetDaemon) StartSubService(requestInfo *skynet.RequestInfo, in M, out *M) (err error) {
	*out = map[string]interface{}{}
	uuid, ok := in.String("uuid")
	if !ok {
		err = errors.New("No UUID provided")
		return
	}

	ss := s.getSubService(uuid)
	if ss != nil {
		ss.Start()
	} else {
		err = errors.New(fmt.Sprintf("No such service UUID %q", uuid))
	}
	return
}

func (s *SkynetDaemon) StopSubService(requestInfo *skynet.RequestInfo, in M, out *M) (err error) {
	*out = map[string]interface{}{}
	uuid, ok := in.String("uuid")
	if !ok {
		err = errors.New("No UUID provided")
		return
	}
	ss := s.getSubService(uuid)
	ss.Stop()
	return
}

func (s *SkynetDaemon) RestartSubService(requestInfo *skynet.RequestInfo, in M, out *M) (err error) {
	*out = map[string]interface{}{}
	uuid, ok := in.String("uuid")
	if !ok {
		err = errors.New("No UUID provided")
		return
	}
	ss := s.getSubService(uuid)
	ss.Restart()
	return
}
