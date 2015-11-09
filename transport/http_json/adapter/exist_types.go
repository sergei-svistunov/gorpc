package adapter

type StringsStack []string

func (ss *StringsStack) Add(s string) {
	*ss = append(*ss, s)
}
func (ss *StringsStack) AlreadyExist(s string) bool {
	for i := range *ss {
		if s == (*ss)[i] {
			return true
		}
	}
	return false
}
