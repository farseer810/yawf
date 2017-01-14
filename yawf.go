package yawf

import (
	"errors"
	"github.com/codegangsta/inject"
	"log"
	"net"
	"net/http"
	"os"
)

type Handler interface{}

type YawfServer interface {
	Router
	Use(Handler)
	SetAddress(address string)
	Address() string
	SetListener(net.Listener)
	Listener() net.Listener
	Listen() error
	Run() error
	RunOnAddress(string) error

	SetLogger(*log.Logger)
	Logger() *log.Logger
}

type yawf struct {
	inject.Injector
	handlers []Handler
	action   Handler
	listener *net.Listener
	logger   *log.Logger
	address  string
}

type classicYawf struct {
	*yawf
	Router
}

const (
	DEFAULT_PORT_ENV_NAME = "YAWF_PORT_ENV_NAME"
	DEFAULT_HOST_ENV_NAME = "YAWF_HOST_ENV_NAME"
)

func New() YawfServer {
	r := NewRouter()
	y := &yawf{Injector: inject.New(), logger: log.New(os.Stdout, "[yawf] ", 0), action: func() {}}
	y.SetLogger(y.logger)
	y.Map(defaultRouterReturnHandler())
	y.Map(defaultMiddlewareReturnHandler())
	y.SetAction(r.Handle)
	return &classicYawf{y, r}
}

func (s *yawf) Listen() error {
	listener, err := net.Listen("tcp", s.Address())
	s.SetListener(listener)
	return err
}

func (s *yawf) Run() error {
	if s.listener == nil {
		s.Logger().Fatalln("failed to run server before listening")
		return errors.New("failed to run server before listening")
	}
	server := &http.Server{Addr: s.Address(), Handler: s}
	return server.Serve(s.Listener())
}

func (s *yawf) RunOnAddress(address string) error {
	s.SetAddress(address)
	err := s.Listen()
	if err != nil {
		return err
	}
	return s.Run()
}

func (s *yawf) Use(handler Handler) {
	ValidateHandler(handler)
	s.handlers = append(s.handlers, handler)
}

func (s *yawf) SetAction(handler Handler) {
	ValidateHandler(handler)
	s.action = handler
}

func (s *yawf) SetAddress(address string) {
	s.address = address
}

func (s *yawf) Address() string {
	if s.address == "" {
		port := os.Getenv(DEFAULT_PORT_ENV_NAME)
		if port == "" {
			port = "3000"
		}
		host := os.Getenv(DEFAULT_HOST_ENV_NAME)
		s.address = host + ":" + port
	}
	return s.address
}

func (s *yawf) SetListener(listener net.Listener) {
	s.listener = &listener
}

func (s *yawf) Listener() net.Listener {
	return *s.listener
}

func (s *yawf) SetLogger(logger *log.Logger) {
	s.logger = logger
	s.Map(s.logger)
}

func (s *yawf) Logger() *log.Logger {
	return s.logger
}

// ServeHTTP is the HTTP Entry point for a yawf instance. Useful if you want to control your own HTTP server.
func (s *yawf) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	s.CreateContext(res, req).Next()
}

func (s *yawf) CreateContext(res http.ResponseWriter, req *http.Request) Context {
	c := NewContext(s.handlers, s.action, res)
	c.SetParent(s)
	c.Map(req)
	return c
}
