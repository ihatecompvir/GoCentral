package jsonproto

import (
	"fmt"
	"rb3server/protocols/jsonproto/marshaler"
	"rb3server/protocols/jsonproto/services/config"
	"rb3server/protocols/jsonproto/services/setlistcreation"
)

type Service interface {
	// the unique path to lookup the service by
	Path() string

	// function to process request
	Handle(string) (string, error)
}

type ServicesManager struct {
	services map[string]Service
}

// Creates new services manager
func NewServicesManager() *ServicesManager {
	mgr := &ServicesManager{
		services: make(map[string]Service),
	}

	// register all services
	mgr.registerAll()

	return mgr

}

// register all services
func (mgr *ServicesManager) registerAll() {
	// config
	mgr.register(config.ConfigService{})

	// setlist creation
	mgr.register(setlistcreation.SetlistCreationService{})

}

// register a single service
func (mgr *ServicesManager) register(service Service) {
	mgr.services[service.Path()] = service
}

// delegates the request to the proper service
func (mgr ServicesManager) Handle(jsonStr string) (string, error) {

	methodPath, err := marshaler.GetRequestName(jsonStr)
	if err != nil {
		return "", err
	}

	// check service is implemented
	service, exists := mgr.services[methodPath]
	if !exists {
		return "", fmt.Errorf("unimplemented service for path:%s\n", methodPath)
	}

	return service.Handle(jsonStr)

}
