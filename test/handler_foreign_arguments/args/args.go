package args

type V1Args struct {
	ReqInt int  `key:"req_int" description:"Required integer argument"`
	Int    *int `key:"int" description:"Unrequired integer argument"`
}
