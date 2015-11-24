package handler_foreign_return_values

import (
	return_values "github.com/sergei-svistunov/gorpc/test/handler_foreign_return_values/return_values"
	"golang.org/x/net/context"
)

type v1Args struct {
	ReqInt int  `key:"req_int" description:"Required integer argument"`
	Int    *int `key:"int" description:"Unrequired integer argument"`
}

type V1Res struct {
	intField    int
	structField *return_values.V1Res
}

func (*Handler) V1(ctx context.Context, opts *v1Args) ([]*V1Res, error) {
	result := []*V1Res{
		{
			structField: &return_values.V1Res{
				String: "Test",
				Int:    opts.ReqInt,
			},
		},
	}
	return result, nil
}
