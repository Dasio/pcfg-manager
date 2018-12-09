package manager

import (
	"sync"
	"sync/atomic"
)

type Generator struct {
	pcfg       *Pcfg
	pQue       *PcfqQueue
	goRoutines int
	generated  uint64
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

func (g *Generator) Run(goRoutines uint, maxGuesses uint64) error {
	if goRoutines <= 0 {
		goRoutines = 1
	}
	var err error
	var item *QueueItem
	jobs := make(chan *TreeItem, goRoutines*64)
	wg := sync.WaitGroup{}
	wg.Add(int(goRoutines))
	for w := uint(1); w <= goRoutines; w++ {
		go func() {
			g.worker(jobs)
			wg.Done()
		}()
	}

	for err != ErrPriorirtyQueEmpty {
		if maxGuesses > 0 && g.generated >= maxGuesses {
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
		jobs <- item.Tree
	}
	close(jobs)
	wg.Wait()
	return nil
}
