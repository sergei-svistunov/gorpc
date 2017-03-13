package handler_common_type_in_different_versions_arguments

import (
	"context"
)

type v1Args struct {
	Field CommonStruct1 `key:"common_type" description:"Field with shared type between handlers"`
}

type V1Res struct {
	String string `json:"string" description:"String field"`
	Int    int    `json:"int" description:"Int field"`
}

func (*Handler) V1(ctx context.Context, opts *v1Args) (*V1Res, error) {
	return &V1Res{
		String: "Test",
		Int:    1,
	}, nil
}
