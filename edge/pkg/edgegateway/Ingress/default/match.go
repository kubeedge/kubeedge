package defaults

import "reflect"

type equalFunction func(e1, e2 interface{}) bool

// Compare checks if the parameters are iterable and contains the same elements
func Compare(listA, listB interface{}, eq equalFunction) bool {
	ok := isIterable(listA)
	if !ok {
		return false
	}

	ok = isIterable(listB)
	if !ok {
		return false
	}

	a := reflect.ValueOf(listA)
	b := reflect.ValueOf(listB)

	if a.IsNil() && b.IsNil() {
		return true
	}

	if a.IsNil() != b.IsNil() {
		return false
	}

	if a.Len() != b.Len() {
		return false
	}

	visited := make([]bool, b.Len())

	for i := 0; i < a.Len(); i++ {
		found := false
		for j := 0; j < b.Len(); j++ {
			if visited[j] {
				continue
			}

			if eq(a.Index(i).Interface(), b.Index(j).Interface()) {
				visited[j] = true
				found = true
				break
			}
		}

		if !found {
			return false
		}
	}

	return true
}

var compareStrings = func(e1, e2 interface{}) bool {
	s1, ok := e1.(string)
	if !ok {
		return false
	}

	s2, ok := e2.(string)
	if !ok {
		return false
	}

	return s1 == s2
}

// StringElementsMatch compares two string slices and returns if are equals
func StringElementsMatch(a, b []string) bool {
	return Compare(a, b, compareStrings)
}

func isIterable(obj interface{}) bool {
	switch reflect.TypeOf(obj).Kind() {
	case reflect.Slice, reflect.Array:
		return true
	default:
		return false
	}
}

