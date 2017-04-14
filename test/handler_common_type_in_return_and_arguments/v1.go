package handler_common_type_in_different_versions_return_values

import (
	"context"
)

type v1Args struct {
	ReqInt     int        `key:"req_int" description:"Required integer argument"`
	Int        *int       `key:"int" description:"Unrequired integer argument"`
	CommonType CommonType `key:"common_struct" description:"Common shared struct"`
}

type V1Res struct {
	String     string      `json:"string" description:"String field"`
	Int        int         `json:"int" description:"Int field"`
	CommonType *CommonType `json:"common_struct" description:"Common shared struct"`
}

type CommonType struct {
	Name string `key:"name" json:"name" description:"String field"`
}

func (*Handler) V1(ctx context.Context, opts *v1Args) (*V1Res, error) {
	return &V1Res{
		String:     "Test",
		Int:        opts.ReqInt,
		CommonType: &CommonType{"Mr. " + opts.CommonType.Name},
	}, nil
}
