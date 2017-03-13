package handler_common_type_in_different_versions_arguments

import (
	"context"
)

type V3Res struct {
	String string `json:"string" description:"String field"`
	Int    int    `json:"int" description:"Int field"`
}

func (*Handler) V3(ctx context.Context, opts *CommonSubStruct) (*V3Res, error) {
	return &V3Res{
		String: "Test",
		Int:    1,
	}, nil
}
