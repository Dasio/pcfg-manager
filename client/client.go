package client

import (
	"context"
	"errors"
	"fmt"
	"github.com/dasio/pcfg-manager/manager"
	pb "github.com/dasio/pcfg-manager/proto"
	"google.golang.org/grpc"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Service struct {
	c           pb.PCFGClient
	mng         *manager.Manager
	grpcConn    *grpc.ClientConn
	grammar     *manager.Grammar
	hashFile    string
	hashcatPath string
	hashcatPipe io.WriteCloser
	hashcatDone chan bool
	// tmp
	hashes []string
}

func NewService(hashcatFolder string) (*Service, error) {
	path, err := filepath.Abs(hashcatFolder + "/" + getHashcatBinary())
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, err
	}
	svc := &Service{
		hashcatPath: path,
		hashcatDone: make(chan bool, 1),
	}
	return svc, nil
}

func getHashcatBinary() string {
	var ext string
	if runtime.GOOS == "windows" {
		ext = "exe"
	} else {
		ext = "bin"
	}
	arch := "32"
	if strings.HasSuffix(runtime.GOARCH, "64") {
		arch = "64"
	}
	return fmt.Sprintf("hashcat%s.%s", arch, ext)
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
	f, err := ioutil.TempFile("", "pcfg-*.hash")
	if err != nil {
		return err
	}
	// tmp
	s.hashes = r.HashList
	s.hashFile = f.Name()
	if _, err := f.Write([]byte(strings.Join(r.HashList, "\n"))); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if _, err := os.Stat(f.Name()); os.IsNotExist(err) {
		return err
	}

	cmd := exec.Command(s.hashcatPath, "-m", r.HashcatMode, "--force", "-O", "-o", "results.txt", "-w", "1", f.Name())
	cmd.Stdout = os.Stdout
	pipe, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	s.hashcatPipe = pipe
	go func() {
		if err := cmd.Run(); err != nil {
			log.Fatal(err)
		}
		fmt.Println("HASHCAT DONE")
		s.hashcatDone <- true
	}()

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
				_, _ = s.c.SendResult(ctx, &pb.CrackingResponse{})
				return err
			}
			cancel()
			if len(res.Items) == 0 {
				_, err = s.c.SendResult(context.Background(), &pb.CrackingResponse{})
				if err != nil {
					return err
				}
				<-s.hashcatDone
				return nil
			}
			for _, item := range res.Items {
				treeItem := pb.TreeItemFromProto(item)
				//s.mng.Generator.Pcfg.ListTerminals(treeItem)
				err := s.mng.Generator.Pcfg.ListTerminalsToWriter(treeItem, s.hashcatPipe)
				if err != nil {
					return err
				}
			}
			resultRes, err := s.c.SendResult(context.Background(), &pb.CrackingResponse{
				Hashes: s.randomResult(),
			})
			if err != nil {
				return err
			}
			if resultRes.End {
				<-s.hashcatDone
				return nil
			}

		}
	}
}

func (s *Service) randomResult() map[string]string {
	if rand.Float32() > 0.8 {
		return map[string]string{
			s.hashes[rand.Int()%len(s.hashes)]: "PassWord123",
		}
	}
	return map[string]string{}
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
	_ = os.Remove(s.hashFile)
	return s.grpcConn.Close()
}
