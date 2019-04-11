package manager

import (
	"sync"
	"sync/atomic"
)

type Generator struct {
	generated  uint64
	pcfg       *Pcfg
	pQue       *PcfqQueue
	goRoutines int
	args       *InputArgs
}

func NewGenerator(pcfg *Pcfg) *Generator {
	que, err := NewPcfgQueue(pcfg)
	if err != nil {
		panic(err)
	}
	return &Generator{
		pcfg: pcfg,
		pQue: que,
	}
}

func (g *Generator) worker(jobs <-chan *TreeItem) {
	for j := range jobs {
		guesses, _, _ := g.pcfg.ListTerminals(j)
		atomic.AddUint64(&g.generated, guesses)
	}

}

func (g *Generator) debugger() {

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
		//guesses, _, _ := g.pcfg.ListTerminals(item.Tree)
		//g.generated += guesses
		jobs <- item.Tree
	}
	close(jobs)
	wg.Wait()

	return nil
}
