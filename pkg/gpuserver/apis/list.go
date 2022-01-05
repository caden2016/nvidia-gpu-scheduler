package apis

func NewList(size int) *List {
	return &List{
		TypeMeta: TypeMeta{Kind: "List", APIVersion: "v1"},
		Items:    make([]interface{}, 0, size),
	}
}

type TypeMeta struct {
	Kind       string `json:"kind,omitempty"`
	APIVersion string `json:"apiVersion,omitempty"`
}

type List struct {
	TypeMeta
	Items []interface{} `json:"items"`
}
