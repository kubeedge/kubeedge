package swagger

// ApiDeclarationList maintains an ordered list of ApiDeclaration.
type ApiDeclarationList struct {
	List []APIDefinition
}

// At returns the ApiDeclaration by its path unless absent, then ok is false
func (l *ApiDeclarationList) At(path string) (a APIDefinition, ok bool) {
	for _, each := range l.List {
		if each.BasePath == path {
			return each, true
		}
	}
	return a, false
}

// Put adds or replaces a ApiDeclaration with this name
func (l *ApiDeclarationList) Put(path string, a APIDefinition) {
	// maybe replace existing
	for i, each := range l.List {
		if each.BasePath == path {
			// replace
			l.List[i] = a
			return
		}
	}
	// add
	l.List = append(l.List, a)
}

// Do enumerates all the properties, each with its assigned name
func (l *ApiDeclarationList) Do(block func(path string, decl APIDefinition)) {
	for _, each := range l.List {
		block(each.BasePath, each)
	}
}
