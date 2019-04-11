package server

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/dasio/pcfg-manager/manager"
	pb "github.com/dasio/pcfg-manager/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type Service struct {
	mng             *manager.Manager
	args            manager.InputArgs
	remainingHashes map[string]struct{}
	completedHashes map[string]string
	generatorCh     <-chan *manager.TreeItem
	priorityCh      chan *manager.TreeItem
	ch              <-chan *manager.TreeItem
	clients         map[string]ClientInfo
	chunkId         uint32
	endCracking     chan bool
}

type Chunk struct {
	Id             uint32
	Items          []*pb.TreeItem
	TerminalsCount uint64
}

func NewService() *Service {
	return &Service{
		clients: make(map[string]ClientInfo),
		chunkId: 0,
	}
}

type ClientInfo struct {
	Addr              string
	ActualChunk       Chunk
	StartTime         time.Time
	EndTime           time.Time
	PreviousTerminals uint64
}

func (s *Service) Load(args manager.InputArgs) error {
	s.args = args
	lines, err := readLines(s.args.HashFile)
	if err != nil {
		return err
	}
	s.remainingHashes = make(map[string]struct{})
	s.completedHashes = make(map[string]string)
	s.endCracking = make(chan bool)
	for _, l := range lines {
		s.remainingHashes[l] = struct{}{}
	}
	s.mng = manager.NewManager(s.args.RuleName)
	if err := s.mng.Load(); err != nil {
		return err
	}
	s.generatorCh = s.mng.Generator.RunForServer(&args)
	s.priorityCh = make(chan *manager.TreeItem)
	s.ch = mergeChannels(s.generatorCh, s.priorityCh)
	return nil
}
func (s *Service) Run() error {
	lis, err := net.Listen("tcp", ":"+s.args.Port)
	if err != nil {
		return err
	}
	server := grpc.NewServer()
	pb.RegisterPCFGServer(server, s)
	logrus.Infof("Listening on port %s", s.args.Port)
	go func() {
		<-s.endCracking
		server.GracefulStop()
	}()
	if err := server.Serve(lis); err != nil {
		return err
	}
	for hash, pass := range s.completedHashes {
		fmt.Println(hash, " ", pass)
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
	var hashList []string
	for k := range s.remainingHashes {
		hashList = append(hashList, k)
	}
	logrus.Infof("client %s connected", client.Addr)
	return &pb.ConnectResponse{
		Grammar:     pb.GrammarToProto(s.mng.Generator.Pcfg.Grammar),
		HashList:    hashList,
		HashcatMode: s.args.HashcatMode,
	}, nil
}

func (s *Service) Disconnect(ctx context.Context, req *pb.Empty) (*pb.Empty, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return &pb.Empty{}, errors.New("no peer")
	}
	clientInfo, ok := s.clients[p.Addr.String()]
	if !ok {
		return &pb.Empty{}, errors.New("client wasn't connected")
	}
	if clientInfo.ActualChunk.Id != 0 {
		logrus.Infof("client %s did not finished chunk[%d], sending %d preterminals back to channel",
			clientInfo.Addr, clientInfo.ActualChunk.Id, len(clientInfo.ActualChunk.Items))
		for _, it := range clientInfo.ActualChunk.Items {
			s.priorityCh <- pb.TreeItemFromProto(it)
		}
	}
	delete(s.clients, p.Addr.String())
	logrus.Infof("client %s disconnected", p.Addr.String())

	return &pb.Empty{}, nil
}

func (s *Service) GetNextChunk(size uint64) (Chunk, bool) {
	total := uint64(0)
	endGen := false
	var chunkItems []*pb.TreeItem
loop:
	for total < size {
		select {
		case it := <-s.ch:
			if it == nil {
				endGen = true
				break loop
			}
			chunkItems = append(chunkItems, pb.TreeItemToProto(it))
			guessGeneration := manager.NewGuessGeneration(s.mng.Generator.Pcfg.Grammar, it)
			total += guessGeneration.Count()
		case <-time.After(time.Second * 2):
			break loop
		}
	}
	return Chunk{
		Id:             atomic.AddUint32(&s.chunkId, 1),
		Items:          chunkItems,
		TerminalsCount: total,
	}, endGen
}

func mergeChannels(cs ...<-chan *manager.TreeItem) <-chan *manager.TreeItem {
	out := make(chan *manager.TreeItem)
	var wg sync.WaitGroup
	wg.Add(len(cs))
	for _, c := range cs {
		go func(c <-chan *manager.TreeItem) {
			for v := range c {
				out <- v
			}
			wg.Done()
		}(c)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}
func (s *Service) GetNextItems(ctx context.Context, req *pb.Empty) (*pb.TreeItems, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return &pb.TreeItems{}, errors.New("no peer")
	}
	then := time.Now()
	clientInfo := s.clients[p.Addr.String()]
	chunkSize := s.args.ChunkStartSize
	if !clientInfo.EndTime.IsZero() && clientInfo.PreviousTerminals != 0 {
		speed := float64(clientInfo.PreviousTerminals) / clientInfo.EndTime.Sub(clientInfo.StartTime).Seconds()
		chunkSize = uint64(speed * s.args.ChunkDuration.Seconds())
	}
	chunk, endGen := s.GetNextChunk(chunkSize)
	if endGen && len(chunk.Items) == 0 {
		return &pb.TreeItems{}, nil
	}
	clientInfo.ActualChunk = chunk
	clientInfo.StartTime = time.Now()
	s.clients[p.Addr.String()] = clientInfo
	logrus.Infof("sending chunk[%d], preTerminals: %d, terminals: %d to %s in %s",
		chunk.Id, len(chunk.Items), chunk.TerminalsCount, clientInfo.Addr, time.Now().Sub(then).String())
	return &pb.TreeItems{
		Items: chunk.Items,
	}, nil
}

func (s *Service) SendResult(ctx context.Context, in *pb.CrackingResponse) (*pb.ResultResponse, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return &pb.ResultResponse{End: false}, errors.New("no peer")
	}
	clientInfo := s.clients[p.Addr.String()]
	for hash, password := range in.Hashes {
		delete(s.remainingHashes, hash)
		s.completedHashes[hash] = password
	}
	clientInfo.EndTime = time.Now()
	clientInfo.PreviousTerminals = clientInfo.ActualChunk.TerminalsCount
	clientInfo.ActualChunk = Chunk{}
	s.clients[p.Addr.String()] = clientInfo

	logrus.Infof("result from %s: %d in %f seconds", clientInfo.Addr, len(in.Hashes), clientInfo.EndTime.Sub(clientInfo.StartTime).Seconds())
	if len(s.remainingHashes) == 0 {
		s.endCracking <- true
		return &pb.ResultResponse{End: true}, nil
	}
	return &pb.ResultResponse{End: false}, nil
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}
