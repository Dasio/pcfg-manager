package manager

type Manager struct {
	generator *Generator
	ruleName  string
}

func NewManager(ruleName string) *Manager {
	return &Manager{
		ruleName: ruleName,
	}
}

func (m *Manager) Load() error {
	grammar, err := LoadGrammar(m.ruleName)
	if err != nil {
		return err
	}
	pcfg := NewPcfg(grammar)
	m.generator = NewGenerator(pcfg)
	return nil
}

func (m *Manager) Start(goRoutines uint, maxGuesess uint64) error {
	if err := m.generator.Run(goRoutines, maxGuesess); err != nil {
		return err
	}
	return nil
}
