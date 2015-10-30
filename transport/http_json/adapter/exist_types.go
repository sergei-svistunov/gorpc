package adapter

type ExistStructs []string

func (es *ExistStructs) Add(s string) {
    *es = append(*es, s)
}
func (es *ExistStructs) AlreadyExist(s string) bool {
    for i := range *es {
        if s == (*es)[i] {
            return true
        }
    }
    return false
}