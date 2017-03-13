package handler_common_type_in_different_versions_arguments

import (
	"context"
)

type V4Res struct {
	String string `json:"string" description:"String field"`
	Int    int    `json:"int" description:"Int field"`
}

func (*Handler) V4(ctx context.Context, opts *CommonSubStruct) (*V4Res, error) {
	return &V4Res{
		String: "Test",
		Int:    1,
	}, nil
}
