package manager

import (
	"bufio"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"gopkg.in/ini.v1"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

const (
	RulesFolder = "Rules/"
)

var (
	rgxPythArr = regexp.MustCompile(`"(.*?)"`)
)

type GrammarMapping map[string]map[string]int

type Grammar struct {
	cfgFile     *ini.File
	ruleName    string
	sectionList []string
	Sections    []*Section
	Mapping     GrammarMapping
}

type ConfigReplacement struct {
	TransitionId string `json:"Transition_id"`
	ConfigId     string `json:"Config_id"`
}

type Replacement struct {
	Probability float64
	IsTerminal  bool
	Values      []string
	Function    string
	Pos         []int
}

type InputBase struct {
	Base        string
	Probability float64
}

type Section struct {
	Type         string
	Name         string
	Replacements []*Replacement
}

func LoadGrammar(ruleName string) (*Grammar, error) {
	grammar := &Grammar{
		ruleName: ruleName,
	}
	if err := grammar.Parse(); err != nil {
		return nil, err
	}
	return grammar, nil
}

func (g *Grammar) Parse() error {
	var err error
	g.sectionList = []string{}
	g.cfgFile, err = ini.Load(RulesFolder + g.ruleName + "/" + "config.ini")
	if err != nil {
		return err
	}
	g.Mapping = make(GrammarMapping)
	logrus.Info("Config file loaded")
	if err := g.Build("START"); err != nil {
		return err
	}
	return nil
}

func (g *Grammar) Build(section string) error {
	for _, s := range g.sectionList {
		if s == section {
			return nil
		}
	}
	g.sectionList = append(g.sectionList, section)

	isTerminal, err := g.cfgFile.Section(section).Key("is_terminal").Bool()
	if err != nil {
		return err
	}
	if !isTerminal {
		var replacements []ConfigReplacement
		b := []byte(g.cfgFile.Section(section).Key("replacements").String())
		if err := json.Unmarshal(b, &replacements); err != nil {
			return err
		}
		for _, replace := range replacements {
			if err := g.Build(replace.ConfigId); err != nil {
				return err
			}
		}
		if err := g.FindGrammarMapping(section); err != nil {
			return err
		}
		err = g.InsertTerminal(section)

	} else {
		err = g.InsertTerminal(section)
	}
	if err != nil {
		return err
	}
	return nil
}
func (g *Grammar) GetGrammarPos(id, length string) (int, error) {
	if val1, ok := g.Mapping[id]; ok {
		if val2, ok := val1[length]; ok {
			return val2, nil
		} else {
			return 0, ErrGrammarMapping
		}
	} else {
		return 0, ErrGrammarMapping
	}
}
func (g *Grammar) ParseBaseStructure(base string) ([]int, error) {
	var pos []int
	if base == "M" {
		val, err := g.GetGrammarPos("M", "markov_prob")
		if err != nil {
			return pos, err
		}
		return []int{val}, nil
	}
	// base => elems
	// A111B22C1 => [A111, B22, C1]
	var elems []string
	l := 0
	for s := base; s != ""; s = s[l:] {
		l = strings.IndexFunc(s[1:], unicode.IsUpper) + 1
		if l <= 0 {
			l = len(s)
		}
		elems = append(elems, s[:l])
	}
	for _, e := range elems {
		value := e[:1]
		size := e[1:]
		val, err := g.GetGrammarPos(value, size)
		if err != nil {
			return pos, err
		}
		pos = append(pos, val)
	}
	return pos, nil
}

func fromPythonArray(in string) []string {
	var ret []string
	find := rgxPythArr.FindAllStringSubmatch(in, -1)
	for _, f := range find {
		if len(f) > 1 {
			ret = append(ret, f[1])
		}
	}
	return ret
}

func extractProbability(fileName string) ([]*InputBase, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return []*InputBase{}, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var res []*InputBase
	for scanner.Scan() {
		splitted := strings.Split(scanner.Text(), "\t")
		if len(splitted) != 2 {
			return res, ErrExtractProbability
		}
		prob, err := strconv.ParseFloat(splitted[1], 64)
		if err != nil {
			return res, err
		}
		res = append(res, &InputBase{
			Base:        splitted[0],
			Probability: prob,
		})
	}

	if err := scanner.Err(); err != nil {
		return []*InputBase{}, err
	}
	return res, nil
}

func (g *Grammar) FindGrammarMapping(section string) error {
	var replacements []ConfigReplacement
	b := []byte(g.cfgFile.Section(section).Key("replacements").String())
	if err := json.Unmarshal(b, &replacements); err != nil {
		return err
	}
	for _, replace := range replacements {
		g.Mapping[replace.TransitionId] = make(map[string]int)
		for i, sec := range g.Sections {
			if replace.ConfigId == sec.Type {
				g.Mapping[replace.TransitionId][sec.Name] = i
			}
		}
	}
	return nil
}
func (g *Grammar) InsertTerminal(section string) error {
	function := g.cfgFile.Section(section).Key("function").String()
	isTerminal, err := g.cfgFile.Section(section).Key("is_terminal").Bool()
	if err != nil {
		return err
	}
	filenames := fromPythonArray(g.cfgFile.Section(section).Key("filenames").String())
	directory := RulesFolder + g.ruleName + "/" + g.cfgFile.Section(section).Key("directory").String()

	for _, curFile := range filenames {
		filePath := directory + "/" + curFile
		probabilities, err := extractProbability(filePath)
		if err != nil {
			return err
		}
		curSection := &Section{
			Name: strings.TrimSuffix(curFile, ".txt"),
			Type: section,
		}
		var replacementPos int
		if function == "Capitalization" || function == "Copy" || function == "Shadow" || function == "Markov" {
			curReplacement := &Replacement{
				Function:    function,
				IsTerminal:  isTerminal,
				Probability: probabilities[0].Probability,
				Values:      []string{probabilities[0].Base},
			}
			if function == "Shadow" {
				var replacements []ConfigReplacement
				b := []byte(g.cfgFile.Section(section).Key("replacements").String())
				if err := json.Unmarshal(b, &replacements); err != nil {
					return err
				}
				val, err := g.GetGrammarPos(replacements[0].TransitionId, curSection.Name)
				if err != nil {
					return err
				}
				replacementPos = val
				curReplacement.Pos = []int{replacementPos}
			}
			lastProb := probabilities[0].Probability
			for i := 1; i < len(probabilities); i++ {

				if probabilities[i].Probability == lastProb {
					curReplacement.Values = append(curReplacement.Values, probabilities[i].Base)
				} else if probabilities[i].Probability < lastProb {
					curSection.Replacements = append(curSection.Replacements, curReplacement)
					lastProb = probabilities[i].Probability
					curReplacement = &Replacement{
						Function:    function,
						IsTerminal:  isTerminal,
						Probability: probabilities[i].Probability,
						Values:      []string{probabilities[i].Base},
					}

					if function == "Shadow" {
						curReplacement.Pos = []int{replacementPos}
					}
				} else {
					return ErrOrderList
				}
			}
			curSection.Replacements = append(curSection.Replacements, curReplacement)
			g.Sections = append(g.Sections, curSection)

		} else if function == "Transparent" {
			for i := range probabilities {
				curReplacement := &Replacement{
					Function:    function,
					IsTerminal:  isTerminal,
					Probability: probabilities[i].Probability,
					Values:      []string{probabilities[i].Base},
				}
				pos, err := g.ParseBaseStructure(probabilities[i].Base)
				if err != nil {
					return err
				}
				curReplacement.Pos = pos
				curSection.Replacements = append(curSection.Replacements, curReplacement)
			}
			g.Sections = append(g.Sections, curSection)
		} else {
			return ErrInvaldiGrammarType
		}
	}
	return nil
}
