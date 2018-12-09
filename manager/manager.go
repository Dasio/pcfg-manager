package manager

import log "github.com/sirupsen/logrus"

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

func (m *Manager) Start(input *InputArgs) error {
	log.Infoln("Rule: ", m.ruleName)
	log.Infoln("GoRoutines: ", input.GoRoutines)
	log.Infoln("MaxGuesses: ", input.MaxGuesses)
	log.Infoln("Debug: ", input.Debug)

	if err := m.generator.Run(input); err != nil {
		return err
	}
	return nil
}
