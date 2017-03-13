package handler_common_type_in_different_versions_return_values

import (
	"context"
)

type v1Args struct {
	ReqInt int  `key:"req_int" description:"Required integer argument"`
	Int    *int `key:"int" description:"Unrequired integer argument"`
}

type V1Res struct {
	String     string           `json:"string" description:"String field"`
	Int        int              `json:"int" description:"Int field"`
	CommonType *CommonSubStruct `json:"common_sub_struct" description:"Common shared struct"`
}

func (*Handler) V1(ctx context.Context, opts *v1Args) (*V1Res, error) {
	return &V1Res{
		String: "Test",
		Int:    opts.ReqInt,
	}, nil
}
