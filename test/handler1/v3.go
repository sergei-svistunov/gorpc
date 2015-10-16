package handler1

import (
	"golang.org/x/net/context"
)

type V3Request struct {
	ReqInt   int         `json:"req_int" key:"req_int" description:"Required integer argument"`
	Nested   V3Nested    `json:"nested" key:"nested" description:"Nested field"`
	Optional *V3Optional `json:"optional" key:"optional" description:"Optional field"`
}

type V3Nested struct {
	ReturnErrorID *int `json:"error_id" key:"error_id" description:"Handler returns error by id if defined"`
}

type V3Optional struct {
	Foo bool `json:"foo" key:"foo" description:"Required bool field"`
}

type V3Response struct {
	Int int   `json:"int" key:"int" description:"Required int field"`
	B   *bool `json:"b,omitempty" key:"b" description:"Optional bool field"`
}

func (*Handler) V3(ctx context.Context, opts *V3Request) (*V3Response, error) {
	resp := &V3Response{
		Int: opts.ReqInt,
	}
	if opts.Optional != nil {
		resp.B = &opts.Optional.Foo
	}
	return resp, nil
}