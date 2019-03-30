package manager

import (
	"github.com/sirupsen/logrus"
	"sync"
	"sync/atomic"
)

type Generator struct {
	Pcfg       *Pcfg
	pQue       *PcfqQueue
	goRoutines int
	generated  uint64
	args       *InputArgs
}

func NewGenerator(pcfg *Pcfg) *Generator {
	que, err := NewPcfgQueue(pcfg)
	if err != nil {
		panic(err)
	}
	return &Generator{
		Pcfg: pcfg,
		pQue: que,
	}
}

func (g *Generator) worker(jobs <-chan *TreeItem) {
	for j := range jobs {
		guesses, _, _ := g.Pcfg.ListTerminals(j)
		atomic.AddUint64(&g.generated, guesses)
	}

}

func (g *Generator) debugger() {

}

func (g *Generator) RunForServer(args *InputArgs) <-chan *TreeItem {
	ch := make(chan *TreeItem, 100)
	go func() {
		var err error
		var item *QueueItem
		for err != ErrPriorirtyQueEmpty {
			if args.MaxGuesses > 0 && g.generated >= args.MaxGuesses {
				break
			}
			item, err = g.pQue.Next()
			if err != nil {
				if err != ErrPriorirtyQueEmpty {
					logrus.Warn(err)
				}
				break
			}
			ch <- item.Tree
		}
		close(ch)
	}()
	return ch
}

func (g *Generator) Run(args *InputArgs) error {
	g.args = args

	var err error
	var item *QueueItem
	jobs := make(chan *TreeItem, args.GoRoutines)
	wg := sync.WaitGroup{}
	wg.Add(int(args.GoRoutines))

	if args.Debug {
		wg.Add(1)
		go func() {
			g.debugger()
			wg.Done()
		}()
	}
	for w := uint(1); w <= args.GoRoutines; w++ {
		go func() {
			g.worker(jobs)
			wg.Done()
		}()
	}

	for err != ErrPriorirtyQueEmpty {
		if args.MaxGuesses > 0 && g.generated >= args.MaxGuesses {
			break
		}
		item, err = g.pQue.Next()
		if err != nil {
			if err == ErrPriorirtyQueEmpty {
				break
			}
			close(jobs)
			return err
		}
		//guesses, _, _ := g.Pcfg.ListTerminals(item.Tree)
		//g.generated += guesses
		jobs <- item.Tree
	}
	close(jobs)
	wg.Wait()

	return nil
}
