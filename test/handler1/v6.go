package handler1

import (
	"golang.org/x/net/context"
)

type V6Request struct {
	F1 *[][]V6Struct1 `key:"f1" description:"f1"`
}

type V6Struct1 struct {
	F11 *V6Struct2 `key:"f11" description:"f11"`
}

type V6Struct2 struct {
	F111 *string `key:"f111" description:"f111"`
}

func (*Handler) V6(ctx context.Context, opts *V6Request) (int, error) {
	return 0, nil
}
