package handler_common_type_in_different_versions_return_values

import (
	"context"
)

type v5Args struct {
	ReqInt int  `key:"req_int" description:"Required integer argument"`
	Int    *int `key:"int" description:"Unrequired integer argument"`
}

func (*Handler) V5(ctx context.Context, opts *v5Args) (*CommonStruct1, error) {
	return &CommonStruct1{}, nil
}
