package main

import (
	"errors"
	"fmt"
	"github.com/skynetservices/skynet"
	"github.com/skynetservices/skynet/daemon"
	"github.com/skynetservices/skynet/service"
	"sync"
)

// SkynetDaemon is a service for administering other services
type SkynetDaemon struct {
	Log         skynet.SemanticLogger
	Services    map[string]*SubService
	serviceLock sync.Mutex
	Service     *service.Service
}

func (sd *SkynetDaemon) Registered(s *service.Service)   {}
func (sd *SkynetDaemon) Unregistered(s *service.Service) {}
func (sd *SkynetDaemon) Started(s *service.Service)      {}

// TODO: Should we stop all services? how do we account for graceful restarts?
func (sd *SkynetDaemon) Stopped(s *service.Service) {
	sd.StopAllSubServices(&skynet.RequestInfo{}, daemon.StopAllSubServicesRequest{}, &daemon.StopAllSubServicesResponse{})
}

func (s *SkynetDaemon) Deploy(requestInfo *skynet.RequestInfo, in daemon.DeployRequest, out *daemon.DeployResponse) (err error) {
	out.UUID = skynet.UUID()

	s.Log.Trace(fmt.Sprintf("%+v", SubserviceDeployment{
		ServicePath: in.ServicePath,
		Args:        in.Args,
	}))

	ss, err := NewSubService(s, in.ServicePath, in.Args, out.UUID)
	if err != nil {
		return
	}

	s.serviceLock.Lock()
	s.Services[out.UUID] = ss
	s.serviceLock.Unlock()

	start, startErr := ss.Start()

	if startErr != nil {
		return errors.New("Service failed to start: " + startErr.Error())
	} else if !start {
		return errors.New("Service failed to start")
	}

	return
}

func (s *SkynetDaemon) getSubService(uuid string) (ss *SubService) {
	s.serviceLock.Lock()
	ss = s.Services[uuid]
	s.serviceLock.Unlock()
	return
}

func (s *SkynetDaemon) ListSubServices(requestInfo *skynet.RequestInfo, in daemon.ListSubServicesRequest, out *daemon.ListSubServicesResponse) (err error) {
	out.Services = make(map[string]daemon.SubServiceInfo)
	if len(s.Services) == 0 {
		err = errors.New("No services deployed")
		return
	}
	for uuid, ss := range s.Services {
		out.Services[uuid] = daemon.SubServiceInfo{
			UUID:        uuid,
			ServicePath: ss.ServicePath,
			Args:        ss.Args,
			Running:     ss.running,
		}
	}
	fmt.Println(out)
	return
}

func (s *SkynetDaemon) StopAllSubServices(requestInfo *skynet.RequestInfo, in daemon.StopAllSubServicesRequest, out *daemon.StopAllSubServicesResponse) (err error) {
	var uuids []string
	s.serviceLock.Lock()
	for uuid := range s.Services {
		uuids = append(uuids, uuid)
	}
	s.serviceLock.Unlock()

	out.Stops = make([]daemon.StopSubServiceResponse, len(uuids))

	for i, uuid := range uuids {
		err = s.StopSubService(requestInfo, daemon.StopSubServiceRequest{UUID: uuid}, &out.Stops[i])
		if err != nil {
			return
		}
		if out.Stops[i].Ok {
			out.Count++
		}
	}

	return
}

func (s *SkynetDaemon) StopSubService(requestInfo *skynet.RequestInfo, in daemon.StopSubServiceRequest, out *daemon.StopSubServiceResponse) (err error) {
	ss := s.getSubService(in.UUID)
	if ss != nil {
		out.Ok = ss.Stop()
		out.UUID = in.UUID
	} else {
		err = errors.New(fmt.Sprintf("No such service UUID %q", in.UUID))
	}

	return
}

func (s *SkynetDaemon) RestartSubService(requestInfo *skynet.RequestInfo, in daemon.RestartSubServiceRequest, out *daemon.RestartSubServiceResponse) (err error) {
	ss := s.getSubService(in.UUID)
	if ss != nil {
		ss.Restart()
		out.UUID = in.UUID
	} else {
		err = errors.New(fmt.Sprintf("No such service UUID %q", in.UUID))
	}
	return
}

func (s *SkynetDaemon) RestartAllSubServices(requestInfo *skynet.RequestInfo, in daemon.RestartAllSubServicesRequest, out *daemon.RestartAllSubServicesResponse) (err error) {
	var uuids []string
	s.serviceLock.Lock()
	for uuid := range s.Services {
		uuids = append(uuids, uuid)
	}
	s.serviceLock.Unlock()

	if len(uuids) == 0 {
		err = errors.New("No services deployed")
		return
	}

	out.Restarts = make([]daemon.RestartSubServiceResponse, len(uuids))

	for i, uuid := range uuids {
		err = s.RestartSubService(requestInfo, daemon.RestartSubServiceRequest{UUID: uuid}, &out.Restarts[i])
		if err != nil {
			return
		}
	}
	return
}
