package proto

import "github.com/dasio/pcfg-manager/manager"

func GrammarToProto(g *manager.Grammar) *Grammar {
	return &Grammar{
		RuleName: g.RuleName,
		Sections: sectionsToProto(g.Sections),
		Mapping:  mappingToProto(g.Mapping),
	}
}

func GrammarFromProto(g *Grammar) *manager.Grammar {
	return &manager.Grammar{
		RuleName: g.RuleName,
		Sections: sectionsFromProto(g.Sections),
		Mapping:  mappingFromProto(g.Mapping),
	}
}

func TreeItemToProto(i *manager.TreeItem) *TreeItem {
	if i == nil {
		return nil
	}
	childrens := make([]*TreeItem, 0, len(i.Childrens))
	for _, ch := range i.Childrens {
		childrens = append(childrens, TreeItemToProto(ch))
	}
	return &TreeItem{
		Index:      i.Index,
		Transition: i.Transition,
		Id:         i.Id,
		Childrens:  childrens,
	}
}

func TreeItemFromProto(i *TreeItem) *manager.TreeItem {
	if i == nil {
		return nil
	}
	childrens := make([]*manager.TreeItem, 0, len(i.Childrens))
	for _, ch := range i.Childrens {
		childrens = append(childrens, TreeItemFromProto(ch))
	}
	return &manager.TreeItem{
		Index:      i.Index,
		Transition: i.Transition,
		Id:         i.Id,
		Childrens:  childrens,
	}
}
func mappingToProto(m manager.GrammarMapping) map[string]*IntMap {
	res := make(map[string]*IntMap)
	for k, v := range m {
		res[k] = &IntMap{
			Value: v,
		}
	}
	return res
}

func mappingFromProto(m map[string]*IntMap) manager.GrammarMapping {
	res := make(manager.GrammarMapping)
	for k, v := range m {
		if v != nil {
			res[k] = v.Value
		}
	}
	return res
}

func sectionToProto(s *manager.Section) *Section {
	return &Section{
		Type:         s.Type,
		Name:         s.Name,
		Replacements: replacementsToProto(s.Replacements),
	}
}

func sectionFromProto(s *Section) *manager.Section {
	return &manager.Section{
		Type:         s.Type,
		Name:         s.Name,
		Replacements: replacementsFromProto(s.Replacements),
	}
}

func sectionsFromProto(s []*Section) []*manager.Section {
	res := make([]*manager.Section, 0, len(s))
	for _, val := range s {
		res = append(res, sectionFromProto(val))
	}
	return res
}

func sectionsToProto(s []*manager.Section) []*Section {
	res := make([]*Section, 0, len(s))
	for _, val := range s {
		res = append(res, sectionToProto(val))
	}
	return res
}
func replacementToProto(r *manager.Replacement) *Replacement {
	return &Replacement{
		Probability: r.Probability,
		IsTerminal:  r.IsTerminal,
		Values:      r.Values,
		Function:    r.Function,
		Pos:         r.Pos,
	}
}

func replacementFromProto(r *Replacement) *manager.Replacement {
	return &manager.Replacement{
		Probability: r.Probability,
		IsTerminal:  r.IsTerminal,
		Values:      r.Values,
		Function:    r.Function,
		Pos:         r.Pos,
	}
}

func replacementsToProto(r []*manager.Replacement) []*Replacement {
	res := make([]*Replacement, 0, len(r))
	for _, val := range r {
		res = append(res, replacementToProto(val))
	}
	return res
}

func replacementsFromProto(r []*Replacement) []*manager.Replacement {
	res := make([]*manager.Replacement, 0, len(r))
	for _, val := range r {
		res = append(res, replacementFromProto(val))
	}
	return res
}
