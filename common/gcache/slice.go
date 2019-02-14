package gcache

//this is fastest one
func SliceInsert(s []interface{}, i int , e interface{})  []interface{} {
	s = append(s, nil /* use the zero value of the element type */)
	copy(s[i+1:], s[i:])
	s[i] = e
	return s
}

func SliceInsert2(s []interface{}, i int , e interface{})  []interface{} {
	s = append(s[:i], append([]interface{}{e}, s[i:]...)...)
	return s
}


func SliceInsert3(s []interface{}, i int , e interface{})  []interface{} {
	tmp:= append([]interface{}{e}, s[i:]...)
	s = append(s[:i], tmp...)
	return s
}

func SliceInsert4(s []interface{}, i int , e interface{})  []interface{} {
	tmp:= s[:i]
	tmp = append(tmp,e)
	s= append(tmp,s[i:]...)
	return s
}
