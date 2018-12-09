package manager

type Generator struct {
	pcfg *Pcfg
	pQue *PcfqQueue
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
		g.pcfg.ListTerminals(j)
	}
}

func (g *Generator) Run() {
	var err error
	var item *QueueItem
	for err != ErrPriorirtyQueEmpty {
		item, err = g.pQue.Next()
		if err != nil {
			if err == ErrPriorirtyQueEmpty {
				break
			}
			panic(err)
		}

		g.pcfg.ListTerminals(item.Tree)
	}

}
