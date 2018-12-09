package manager

type Manager struct {
	ruleName string
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
	generator := NewGenerator(pcfg)
	generator.Run()
	return nil
}

func (m *Manager) Start() error {
	return nil
}
