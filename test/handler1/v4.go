package handler1

import (
	"context"
)

type V4Request struct {
	SliceInSlice [][]V4Struct `key:"slice_in_slice" description:"Slice in slice field"`
}

type V4Struct struct {
	F1 int  `key:"f1" description:"First field"`
	F2 *int `key:"f2" description:"Second field"`
}

func (*Handler) V4(ctx context.Context, opts *V4Request) (int, error) {
	return 0, nil
}
