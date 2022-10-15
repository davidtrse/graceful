package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/labstack/echo/v4"
)

const (
	tusEndpoint = "files"
	tusParam    = "fileID"
)

type GracefulTUSManager interface {
	StartNewTUS(string)
	DoneTUS(string)
	CanShutdown() bool
	StartReceivingRequest()
	StopReceivingRequest()
	IsReceivingRequest() bool
	EchoMiddleware() echo.MiddlewareFunc
}

type GracefulManager struct {
	running       map[string]bool
	acceptRequest bool
	mu            sync.Mutex
}

func NewShutdownManage() GracefulTUSManager {
	return &GracefulManager{
		mu:      sync.Mutex{},
		running: map[string]bool{},
	}
}

func (s *GracefulManager) StartNewTUS(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fmt.Println("Add id to map")
	s.running[id] = true
	bs, _ := json.Marshal(s.running)
	fmt.Println(string(bs))
}

func (s *GracefulManager) DoneTUS(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fmt.Println("Remove id to map")
	delete(s.running, id)
	bs, _ := json.Marshal(s.running)
	fmt.Println(string(bs))
}

func (s *GracefulManager) CanShutdown() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	fmt.Println("CanShutdown, len(s.running)=", s.running)
	return len(s.running) == 0
}

func (s *GracefulManager) StartReceivingRequest() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.acceptRequest = true
}

func (s *GracefulManager) StopReceivingRequest() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.acceptRequest = false
}

func (s *GracefulManager) IsReceivingRequest() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.acceptRequest
}

func (s *GracefulManager) CanReceiveRequest(isCallTUS bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.acceptRequest && !isCallTUS {
		return false
	}

	return true
}

func (s *GracefulManager) EchoMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			fmt.Println("==========>path:", c.Path())
			isCallTUS := strings.Contains(c.Path(), "files")

			id := c.Param("fileID")
			fmt.Println("==========>fileID:", id)
			fmt.Println("==========>s.CanReceiveRequest(id, isCallTUS):", s.CanReceiveRequest(isCallTUS))
			if s.CanReceiveRequest(isCallTUS) {
				return next(c)
			} else {
				return c.NoContent(http.StatusServiceUnavailable)
			}
		}
	}
}
