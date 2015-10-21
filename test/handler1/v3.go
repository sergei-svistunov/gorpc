package handler1

import "golang.org/x/net/context"

type V3Request struct {
	ReqInt      int                   `json:"req_int" key:"req_int" description:"Required integer argument"`
	Nested      V3Nested              `json:"nested" key:"nested" description:"Nested field"`
	Optional    *V3Optional           `json:"optional,omitempty" key:"optional" description:"Optional field"`
	StringMap   map[string]string     `json:"strings" key:"strings" description:"Map field"`
	StringSlice []string              `json:"slices" key:"slices" description:"Slice field"`
	ObjMap      map[string]V3Optional `json:"obj_map" key:"obj_map" description:"Map field"`
	ObjSlice    []V3Optional          `json:"obj_slice" key:"obj_slice" description:"Slice field"`
}

type V3Nested struct {
	ReturnErrorID *int `json:"error_id,omitempty" key:"error_id" description:"Handler returns error by id if defined"`
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
