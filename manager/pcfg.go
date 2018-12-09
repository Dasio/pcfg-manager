package manager

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Pcfg struct {
	grammar *Grammar
}

func NewPcfg(g *Grammar) *Pcfg {
	return &Pcfg{
		grammar: g,
	}
}

func (p *Pcfg) StartIndex() int {
	if len(p.grammar.Sections) == 0 {
		return -1
	}
	if p.grammar.Sections[len(p.grammar.Sections)-1].Type == "START" {
		return len(p.grammar.Sections) - 1
	}
	for i := range p.grammar.Sections {
		if p.grammar.Sections[i].Type == "START" {
			return i
		}
	}
	return -1
}

func (p *Pcfg) ListTerminals(preTerminal *TreeItem) (uint64, string, string) {
	guessGeneration := NewGuessGeneration(p.grammar, preTerminal)
	guess := guessGeneration.First()
	guesses := uint64(0)
	first := guess
	last := guess
	buf := bufio.NewWriter(os.Stdout)
	for guess != "" {
		//fmt.Println(guess)
		fmt.Fprintln(buf, guess)
		guess = guessGeneration.Next()
		last = guess
		guesses++
	}
	buf.Flush()
	//fmt.Println(guesses)
	return guesses, first, last
}

func PrintChildren(chs []*TreeItem, depth int) {
	if len(chs) < 1 {
		return
	}
	for _, ch := range chs {
		fmt.Fprintln(os.Stderr, strings.Repeat("\t", depth), *ch)
		PrintChildren(ch.Childrens, depth+1)
	}

}
func (p *Pcfg) FindProbability(tree *TreeItem) float64 {
	if len(p.grammar.Sections) <= tree.Index || len(p.grammar.Sections[tree.Index].Replacements) <= tree.Transition {
		panic("wrong indexing in find probability")
	}
	prob := p.grammar.Sections[tree.Index].Replacements[tree.Transition].Probability
	if len(tree.Childrens) > 0 {
		for _, children := range tree.Childrens {
			childProb := p.FindProbability(children)
			prob *= childProb
		}
	}
	return prob
}

func (p *Pcfg) FindIsTerminal(tree *TreeItem) bool {
	if len(tree.Childrens) == 0 {
		if !p.grammar.Sections[tree.Index].Replacements[tree.Transition].IsTerminal {
			return false
		}
	} else {
		for _, children := range tree.Childrens {
			if !p.FindIsTerminal(children) {
				return false
			}
		}
	}
	return true
}

func (p *Pcfg) DeadbeatDad(tree *TreeItem) []*TreeItem {
	childrenList := make([]*TreeItem, 0, 4)
	childrenList = p.DDFindChildren(tree, tree, childrenList)
	return childrenList
}

func (p *Pcfg) DDFindChildren(node, parent *TreeItem, childrenList []*TreeItem) []*TreeItem {
	if len(node.Childrens) == 0 {
		numReplacements := len(p.grammar.Sections[node.Index].Replacements)
		// Takes care of the incrementing if there are children for the current node. Aka(1,2,[]) => (1,3,[])
		if numReplacements > node.Transition+1 {
			// Make this a child node
			node.Transition++
			// An id to identify the calling node as the parent
			node.Id = true
			if p.DDIsMyParent(parent, false) {
				node.Id = false
				childrenList = append(childrenList, parent.Copy())
			} else {
				node.Id = false
			}
			// Replace the parent's value
			node.Transition--
		}
		if !p.grammar.Sections[node.Index].Replacements[0].IsTerminal {
			var newExpansion []*TreeItem
			for _, pos := range p.grammar.Sections[node.Index].Replacements[node.Transition].Pos {
				newExpansion = append(newExpansion, &TreeItem{
					Index:      pos,
					Transition: 0,
				})
			}
			// Make this a child node
			node.Childrens = newExpansion
			node.Id = true
			if p.DDIsMyParent(parent, true) {
				node.Id = false
				childrenList = append(childrenList, parent.Copy())
			} else {
				node.Id = false
			}
			// Replace the parent's value
			node.Childrens = []*TreeItem{}
		}
	} else {
		for _, children := range node.Childrens {
			childrenList = p.DDFindChildren(children, parent, childrenList)
		}
	}
	return childrenList
}

func (p *Pcfg) DDIsMyParent(child *TreeItem, isExpansion bool) bool {
	var curNode *TreeItem
	curParseTree := []*TreeItem{child}
	minDiff := 2.0
	foundOrigParent := false
	emptyListParent := false
	for len(curParseTree) > 0 {
		// Pop
		curNode, curParseTree = curParseTree[len(curParseTree)-1], curParseTree[:len(curParseTree)-1]
		if len(curNode.Childrens) == 0 {
			if curNode.Transition != 0 {
				parentProbDiff := p.grammar.Sections[curNode.Index].Replacements[curNode.Transition-1].Probability -
					p.grammar.Sections[curNode.Index].Replacements[curNode.Transition].Probability
				if parentProbDiff < minDiff {
					if curNode.Id && !isExpansion {
						foundOrigParent = true
					} else if foundOrigParent {
						return false
					}
					minDiff = parentProbDiff
				} else if curNode.Id && !isExpansion {
					return false
				}
			}
		} else {
			emptyListParent = true
			for _, children := range curNode.Childrens {
				if children.Transition != 0 || len(children.Childrens) != 0 {
					emptyListParent = false
					curParseTree = append(curParseTree, children)
				}
			}
			if emptyListParent {
				newExpansionProb := 1.0
				for _, children := range curNode.Childrens {
					newExpansionProb *= p.grammar.Sections[children.Index].Replacements[0].Probability
				}
				parentProbDiff := 1.0 - newExpansionProb
				if parentProbDiff < minDiff {
					if curNode.Id && isExpansion {
						foundOrigParent = true
					} else if foundOrigParent {
						return false
					}
					minDiff = parentProbDiff
				} else if curNode.Id && isExpansion {
					return false
				}
			}
		}
	}
	return true
}
