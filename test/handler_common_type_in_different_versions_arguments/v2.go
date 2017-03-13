package handler_common_type_in_different_versions_arguments

import (
	"context"
)

type v2Args struct {
	Field CommonSubStruct `key:"common_type" description:"Field with shared type between handlers"`
}

type V2Res struct {
	String string `json:"string" description:"String field"`
	Int    int    `json:"int" description:"Int field"`
}

func (*Handler) V2(ctx context.Context, opts *v2Args) (*V2Res, error) {
	return &V2Res{
		String: "Test",
		Int:    1,
	}, nil
}
