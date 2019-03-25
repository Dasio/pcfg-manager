package client

import (
	"context"
	"errors"
	"fmt"
	"github.com/dasio/pcfg-manager/manager"
	pb "github.com/dasio/pcfg-manager/proto"
	"google.golang.org/grpc"
	"time"
)

type Service struct {
	c        pb.PCFGClient
	grpcConn *grpc.ClientConn
	grammar  *manager.Grammar
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Connect(address string) error {
	var err error
	s.grpcConn, err = grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return err
	}
	s.c = pb.NewPCFGClient(s.grpcConn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	r, err := s.c.Connect(ctx, &pb.Empty{})
	if err != nil {
		return err
	}
	s.grammar = pb.GrammarFromProto(r.Grammar)
	fmt.Println(*s.grammar)
	return nil
}

func (s *Service) Disconnect() error {
	if s.grpcConn == nil {
		return errors.New("no active grpc connection")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	if _, err := s.c.Disconnect(ctx, &pb.Empty{}); err != nil {
		return err
	}
	return s.grpcConn.Close()
}
