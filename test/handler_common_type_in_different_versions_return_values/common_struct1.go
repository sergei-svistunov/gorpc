package handler_common_type_in_different_versions_return_values

type CommonStruct1 struct {
	F1 []CommonSubStruct `key:"f1" description:"f1"`
	F2 bool              `key:"f2" description:"f2"`
	F3 interface{}       `key:"f3" description:"f3"`
	F4 map[string]string `key:"f4" description:"f4"`
	F5 []CommonStruct1   `key:"f5" description:"Recurcive type"`
}

type CommonSubStruct struct {
	SubF1 []string `key:"sub_f1" description:"sub f1"`
}
