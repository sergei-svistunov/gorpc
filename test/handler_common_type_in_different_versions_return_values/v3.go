package handler_common_type_in_different_versions_return_values

import (
	"context"
)

type v3Args struct {
	ReqInt int  `key:"req_int" description:"Required integer argument"`
	Int    *int `key:"int" description:"Unrequired integer argument"`
}

type V3Res struct {
	String     string         `json:"string" description:"String field"`
	Int        int            `json:"int" description:"Int field"`
	CommonType *CommonStruct1 `json:"common_struct" description:"Common shared struct"`
}

func (*Handler) V3(ctx context.Context, opts *v3Args) (*V3Res, error) {
	return &V3Res{
		String: "Test",
		Int:    opts.ReqInt,
	}, nil
}
