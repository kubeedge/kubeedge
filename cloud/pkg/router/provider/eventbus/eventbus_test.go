package eventbus

import (
	"fmt"
	"path"
	"testing"
)

func TestPathJoin(t *testing.T) {
	var s1, s2 string
	s1 = fmt.Sprintf("%s/node/%s/%s/%s", "bus", "nodeName", "namespace", "subTopic")
	s2 = path.Join("bus/node", "nodeName", "namespace", "subTopic")
	if s1 != s2 {
		t.Fatalf("expected: %s, actual: %s", s1, s2)
	}

	s1 = fmt.Sprintf("node/%s/%s/%s", "nodeName", "namespace", "subTopic")
	s2 = path.Join("node", "nodeName", "namespace", "subTopic")
	if s1 != s2 {
		t.Fatalf("expected: %s, actual: %s", s1, s2)
	}
}
