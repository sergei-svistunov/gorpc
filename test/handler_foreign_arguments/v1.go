package handler_foreign_arguments

import (
	args "github.com/sergei-svistunov/gorpc/test/handler_foreign_arguments/args"
	"context"
)

type V1Res struct {
	String string `json:"string" description:"String field"`
	Int    int    `json:"int" description:"Int field"`
}

func (*Handler) V1(ctx context.Context, opts *args.V1Args) (*V1Res, error) {
	return &V1Res{
		String: "Test",
		Int:    opts.ReqInt,
	}, nil
}
