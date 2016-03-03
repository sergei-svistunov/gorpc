package handler1

import (
	"golang.org/x/net/context"
)

type V5Request struct {
	SliceOfInt *[]int `key:"slice_of_int" description:"Slice of integers field"`
}

func (*Handler) V5(ctx context.Context, opts *V5Request) (int, error) {
	return (*opts.SliceOfInt)[0], nil
}
