package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"github.com/bketelsen/skynet/skylib"
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
func Daemon(q *skylib.Query, argv []string) {

	config, args := skylib.GetServiceConfigFromFlags(argv...)
	config.Name = "SkynetDaemon"
	config.Version = "1"
	config.Region = "Jersey"

	var err error
	mlogger, err := skylib.NewMongoLogger("localhost", "skynet", "log", config)
	clogger := skylib.NewConsoleLogger(os.Stdout)
	config.Log = skylib.NewMultiLogger(mlogger, clogger)
	if err != nil {
		config.Log.Item("Could not connect to mongo db for logging")
	}

	deployment := &SkynetDaemon{
		Log:      config.Log,
		Services: map[string]*SubService{},
	}

	service := skylib.CreateService(deployment, config)

	// handle panic so that we remove ourselves from the pool in case of catastrophic failure
	defer func() {
		service.Shutdown()
		if err := recover(); err != nil {
			log.Println("Unrecovered error occured: ", err)
		}
	}()

	if len(args) == 2 {
		err := deployConfig(deployment, args[0])
		if err != nil {
			config.Log.Item(err)
		}
	}

	// If we pass false here service will not be Registered
	// we could do other work/tasks by implementing the Started method and calling Register() when we're ready
	service.Start(true)
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
		s.Deploy(servicePath, args)
	}
	return
}

// SkynetDaemon is a service for administering other services
type SkynetDaemon struct {
	Log         skylib.Logger
	Services    map[string]*SubService
	serviceLock sync.Mutex
}

func (s *SkynetDaemon) Registered(service *skylib.Service)   {}
func (s *SkynetDaemon) Unregistered(service *skylib.Service) {}
func (s *SkynetDaemon) Started(service *skylib.Service)      {}
func (s *SkynetDaemon) Stopped(service *skylib.Service) {
	s.StopAllSubServices()
}

func (s *SkynetDaemon) Deploy(servicePath, args string) (uuid string, err error) {
	uuid = skylib.UUID()

	s.Log.Item(SubserviceDeployment{
		ServicePath: servicePath,
		Args:        args,
	})

	ss, err := NewSubService(s.Log, servicePath, args)
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

func (s *SkynetDaemon) ListSubServices() (data string, err error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	s.serviceLock.Lock()
	enc.Encode(s.Services)
	s.serviceLock.Unlock()
	data = buf.String()
	return
}

func (s *SkynetDaemon) StopAllSubServices() (err error) {
	var uuids []string
	s.serviceLock.Lock()
	for uuid := range s.Services {
		uuids = append(uuids, uuid)
	}
	s.serviceLock.Unlock()
	for _, uuid := range uuids {
		err = s.StopSubService(uuid)
		if err != nil {
			return
		}
	}
	return
}

func (s *SkynetDaemon) StartAllSubServices() (err error) {
	var uuids []string
	s.serviceLock.Lock()
	for uuid := range s.Services {
		uuids = append(uuids, uuid)
	}
	s.serviceLock.Unlock()
	for _, uuid := range uuids {
		err = s.StartSubService(uuid)
		if err != nil {
			return
		}
	}
	return
}

func (s *SkynetDaemon) StartSubService(uuid string) (err error) {
	ss := s.getSubService(uuid)
	ss.Start()
	return
}

func (s *SkynetDaemon) StopSubService(uuid string) (err error) {
	ss := s.getSubService(uuid)
	ss.Stop()
	return
}

func (s *SkynetDaemon) RegisterSubService(uuid string) (err error) {
	ss := s.getSubService(uuid)
	ss.Register()
	return
}

func (s *SkynetDaemon) DeregisterSubService(uuid string) (err error) {
	ss := s.getSubService(uuid)
	ss.Deregister()
	return
}

func (s *SkynetDaemon) RestartSubService(uuid string) (err error) {
	ss := s.getSubService(uuid)
	ss.Restart()
	return
}
