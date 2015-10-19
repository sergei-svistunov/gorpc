package handler_foreign_return_values

import (
	return_values "github.com/sergei-svistunov/gorpc/test/handler_foreign_return_values/return_values"
	"golang.org/x/net/context"
)

type v1Args struct {
	ReqInt int  `key:"req_int" description:"Required integer argument"`
	Int    *int `key:"int" description:"Unrequired integer argument"`
}

func (*Handler) V1UseCache() {}

func (*Handler) V1(ctx context.Context, opts *v1Args) (*return_values.V1Res, error) {
	return &return_values.V1Res{
		String: "Test",
		Int:    opts.ReqInt,
	}, nil
}
