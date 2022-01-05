package serverds

import (
	"fmt"
	"k8s.io/apimachinery/pkg/util/sets"
	"strings"
)

func DumpModelSetInfo(modelset map[string]sets.String) string {
	sb := strings.Builder{}
	for k, v := range modelset {
		sb.WriteString(fmt.Sprintf("[model:%s,sets:%v] ", k, v.List()))
	}
	return sb.String()
}
