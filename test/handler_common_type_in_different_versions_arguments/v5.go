package handler_common_type_in_different_versions_arguments

import (
	"context"
)

type V5Res struct {
	String string `json:"string" description:"String field"`
	Int    int    `json:"int" description:"Int field"`
}

func (*Handler) V5(ctx context.Context, opts *CommonStruct1) (*V5Res, error) {
	return &V5Res{
		String: "Test",
		Int:    1,
	}, nil
}
