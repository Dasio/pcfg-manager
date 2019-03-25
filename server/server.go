package server

import (
	"context"
	"errors"
	"github.com/dasio/pcfg-manager/manager"
	pb "github.com/dasio/pcfg-manager/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"net"
	"time"
)

type Service struct {
	port    string
	mng     *manager.Manager
	ch      <-chan *manager.TreeItem
	clients map[string]ClientInfo
}

func NewService() *Service {
	return &Service{
		port:    ":50051",
		clients: make(map[string]ClientInfo),
	}
}

type ClientInfo struct {
	Addr string
}

func (s *Service) Load(ruleName string) error {
	s.mng = manager.NewManager(ruleName)
	if err := s.mng.Load(); err != nil {
		return err
	}
	s.ch = s.mng.Generator.RunForServer(&manager.InputArgs{})
	return nil
}
func (s *Service) Run() error {
	lis, err := net.Listen("tcp", s.port)
	if err != nil {
		return err
	}
	server := grpc.NewServer()
	pb.RegisterPCFGServer(server, s)
	logrus.Infof("Listening on port %s", s.port)
	if err := server.Serve(lis); err != nil {
		return err
	}
	return nil
}

func (s *Service) Connect(ctx context.Context, req *pb.Empty) (*pb.ConnectResponse, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return &pb.ConnectResponse{}, errors.New("no peer")
	}
	client := ClientInfo{
		Addr: p.Addr.String(),
	}
	s.clients[client.Addr] = client
	logrus.Infof("client %s connected", client.Addr)
	return &pb.ConnectResponse{
		Grammar: pb.GrammarToProto(s.mng.Generator.Pcfg.Grammar),
	}, nil
}

func (s *Service) Disconnect(ctx context.Context, req *pb.Empty) (*pb.Empty, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return &pb.Empty{}, errors.New("no peer")
	}
	delete(s.clients, p.Addr.String())
	logrus.Infof("client %s disconnected", p.Addr.String())

	return &pb.Empty{}, nil
}

func (s *Service) GetNextItems(ctx context.Context, req *pb.Empty) (*pb.TreeItems, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return &pb.TreeItems{}, errors.New("no peer")
	}
	_ = p
	var item *manager.TreeItem
	select {
	case it := <-s.ch:
		item = it
	case <-time.After(time.Second * 5):
	}
	if item == nil {
		return &pb.TreeItems{}, nil
	}
	return &pb.TreeItems{
		Items: []*pb.TreeItem{pb.TreeItemToProto(item)},
	}, nil
}
