package client

import (
	"context"
	"errors"
	"github.com/dasio/pcfg-manager/manager"
	pb "github.com/dasio/pcfg-manager/proto"
	"google.golang.org/grpc"
	"time"
)

type Service struct {
	c        pb.PCFGClient
	mng      *manager.Manager
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
	s.mng = manager.NewManager(s.grammar.RuleName)
	s.mng.LoadWithGrammar(s.grammar)
	return nil
}

func (s *Service) Run(done <-chan bool) error {
	for {
		select {
		case <-done:
			return nil
		default:
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			res, err := s.c.GetNextItems(ctx, &pb.Empty{})
			if err != nil {
				cancel()
				return err
			}
			if len(res.Items) == 0 {
				cancel()
				return nil
			}
			for _, item := range res.Items {
				treeItem := pb.TreeItemFromProto(item)
				s.mng.Generator.Pcfg.ListTerminals(treeItem)
			}
			cancel()
		}
	}
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
