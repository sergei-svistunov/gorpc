package handler1

import (
	"errors"

	"golang.org/x/net/context"
)

type v2Args struct {
	ReqInt        int  `key:"req_int" description:"Required integer argument"`
	ReturnErrorID *int `key:"error_id" description:"Handler returns error by id if defined"`
}

type V2Res struct {
	Int int `json:"int" description:"Intfield"`
}

type V2ErrorTypes struct {
	ERROR_TYPE1 error `text:"Error 1 description"`
	ERROR_TYPE2 error `text:"Error 2 description"`
	ERROR_TYPE3 error `text:"Error 3 description"`
}

var v2Errors V2ErrorTypes

func (*Handler) V2ErrorsVar() *V2ErrorTypes {
	return &v2Errors
}

func (*Handler) V2ConsumeFlatRequest() {}

func (*Handler) V2(ctx context.Context, opts *v2Args) (*V2Res, error) {
	if opts.ReturnErrorID == nil {
		return &V2Res{
			Int: opts.ReqInt,
		}, nil
	}

	switch *opts.ReturnErrorID {
	case 1:
		return nil, v2Errors.ERROR_TYPE1
	case 2:
		return nil, v2Errors.ERROR_TYPE2
	case 3:
		return nil, v2Errors.ERROR_TYPE3
	default:
		return nil, errors.New("Unknown error")
	}
}
