package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"shortener/internal/blackbox"
	"syscall"

	"google.golang.org/grpc"

	pbblackbox "shortener/proto/blackbox"
)

func main() {
	s := grpc.NewServer()

	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalln("couldn't start listening for connections. error:", err)
	}

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGTERM,
		syscall.SIGINT,
	)
	defer cancel()

	service, err := blackbox.New(
		blackbox.WithSecret(os.Getenv("BLACKBOX_SECRET")),
	)
	if err != nil {
		log.Fatalln("couldn't instantiate service impl. error:", err)
	}
	pbblackbox.RegisterBlackboxServiceServer(s, service)

	go func() {
		<-ctx.Done()
		s.GracefulStop()
	}()

	if err := s.Serve(lis); err != nil {
		log.Fatalln("fatal serve error:", err)
	}
}
