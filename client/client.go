package client

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/dasio/pcfg-manager/manager"
	pb "github.com/dasio/pcfg-manager/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"io"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Service struct {
	c           pb.PCFGClient
	mng         *manager.Manager
	grpcConn    *grpc.ClientConn
	grammar     *manager.Grammar
	genOnly     bool
	hashFile    string
	hashcatPath string
	hashcatMode string
	hashcatPipe io.WriteCloser
	// tmp
	hashes []string
}

type InputArgs struct {
	ServerAddress string
	HashcatFolder string
	GenOnly       bool
}

const (
	HsCodeGpuWatchdogAlarm    = -2
	HsCodeError               = -1
	HsCodeOk                  = 0
	HsCodeExhausted           = 1
	HsCodeAborted             = 2
	HsCodeAbortedByCheckpoint = 3
	HsCodeAbortedByRune       = 4
)

var (
	ErrFinished = errors.New("server finished cracking")
)

func NewService(inArgs InputArgs) (*Service, error) {
	var path string
	var err error
	if !inArgs.GenOnly {
		path, err = filepath.Abs(inArgs.HashcatFolder + "/" + getHashcatBinary())
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return nil, err
		}
	}
	svc := &Service{
		hashcatPath: path,
		genOnly:     inArgs.GenOnly,
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
	s.hashcatMode = r.HashcatMode
	s.hashFile = f.Name()
	if _, err := f.Write([]byte(strings.Join(r.HashList, "\n"))); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	return nil
}

func (s *Service) startHashcat() (*exec.Cmd, error) {
	cmd := exec.Command(s.hashcatPath, "-m", s.hashcatMode, "-o", "results.txt", "--machine-readable", "--status", s.hashFile)
	//cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	pipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	s.hashcatPipe = pipe
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}

func (s *Service) Run(done <-chan bool) error {
	for {
		select {
		case <-done:
			return nil
		default:
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
			res, err := s.c.GetNextItems(ctx, &pb.Empty{}, grpc.MaxCallRecvMsgSize(math.MaxInt32))
			if err != nil {
				cancel()
				_, _ = s.c.SendResult(ctx, &pb.CrackingResponse{})
				return err
			}
			logrus.Infof("received %d preTerminals", len(res.Items))
			cancel()
			if len(res.Items) == 0 {
				_, err = s.c.SendResult(context.Background(), &pb.CrackingResponse{})
				if err != nil {
					return err
				}
				return nil
			}
			var results map[string]string
			if s.genOnly {
				results, err = s.generateOnly(res.Items)
			} else {
				results, err = s.startCracking(res.Items)
			}
			if err != nil {
				return err
			}
			resultRes, err := s.c.SendResult(context.Background(), &pb.CrackingResponse{
				Hashes: results,
			})
			logrus.Infof("sending %d cracked hashes", len(results))

			if err != nil {
				return err
			}
			if resultRes.End {
				return ErrFinished
			}

		}
	}
}

func (s *Service) worker(jobs <-chan *pb.TreeItem) {
	for j := range jobs {
		treeItem := pb.TreeItemFromProto(j)
		err := s.mng.Generator.Pcfg.ListTerminalsToWriter(treeItem, os.Stdout)
		if err != nil {
			logrus.Warn(err)
		}
	}

}
func (s *Service) generateOnly(preTerminals []*pb.TreeItem) (map[string]string, error) {
	const goRoutines = 4

	jobs := make(chan *pb.TreeItem, goRoutines)
	wg := sync.WaitGroup{}
	wg.Add(goRoutines)
	for w := uint(1); w <= goRoutines; w++ {
		go func() {
			s.worker(jobs)
			wg.Done()
		}()
	}
	for _, item := range preTerminals {
		jobs <- item
	}
	close(jobs)
	wg.Wait()
	return map[string]string{}, nil
}
func (s *Service) startCracking(preTerminals []*pb.TreeItem) (map[string]string, error) {
	cmd, err := s.startHashcat()
	if err != nil {
		return nil, err
	}
	for _, item := range preTerminals {
		treeItem := pb.TreeItemFromProto(item)
		err := s.mng.Generator.Pcfg.ListTerminalsToWriter(treeItem, s.hashcatPipe)
		if err != nil {
			return nil, err
		}
	}
	if err := s.hashcatPipe.Close(); err != nil {
		return nil, err
	}
	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() != HsCodeOk && exitErr.ExitCode() != HsCodeExhausted {
				return nil, err
			}
		}
	}
	results, err := getResults("results.txt")
	if err != nil {
		return nil, err
	}
	return results, nil
}

func getResults(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		if err == os.ErrNotExist {
			return map[string]string{}, nil
		}
		return nil, err
	}
	defer file.Close()

	res := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		split := strings.Split(scanner.Text(), ":")
		if len(split) != 2 {
			continue
		}
		res[split[0]] = split[1]
	}
	return res, scanner.Err()
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	if _, err := s.c.Disconnect(ctx, &pb.Empty{}); err != nil {
		return err
	}
	_ = os.Remove(s.hashFile)
	return s.grpcConn.Close()
}
