package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

type Server struct {
	addr   string
	s      *http.Server
	ctx    context.Context
	cancel context.CancelFunc
}

func NewServer(addr string) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		addr:   addr,
		s:      &http.Server{},
		ctx:    ctx,
		cancel: cancel,
	}
}

func (s *Server) RegisterHandler() {
	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello world!"))
	})
}

func (s *Server) Run() error {
	l, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	g, ctx := errgroup.WithContext(s.ctx)
	g.Go(func() error {
		<-ctx.Done()
		c, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		return s.s.Shutdown(c)
	})
	g.Go(func() error {
		return s.s.Serve(l)
	})
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-c:
				s.cancel()
			}
		}
	})
	log.Print("server start")
	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		s.cancel()
		return err
	}
	return nil
}

func main() {
	svr := NewServer("0.0.0.0:8080")
	svr.RegisterHandler()
	if err := svr.Run(); err != nil {
		log.Print(err)
	}
}
