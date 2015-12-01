package handler1

import (
	"golang.org/x/net/context"
)

type v1Args struct {
	ReqInt int  `key:"req_int" description:"Required integer argument"`
	Int    *int `key:"int" description:"Unrequired integer argument"`
}

type V1Res struct {
	String string `json:"string" description:"String field"`
	Int    int    `json:"int" description:"Int field"`
}

func (*Handler) V1(ctx context.Context, opts *v1Args) (*V1Res, error) {
	return &V1Res{
		String: "Test",
		Int:    opts.ReqInt,
	}, nil
}
