package errs

/*
	A error type for reach pool max connection limit
*/
type MaxActiveConnectionErr struct {
	msg string
}

func (e MaxActiveConnectionErr) Error() string {
	return e.msg
}

func NewMaxActiveConnectionErr(cause string) MaxActiveConnectionErr {
	return MaxActiveConnectionErr{
		msg: cause,
	}
}

func IsMaxActiveConnectionErr(e error) bool {
	_, ok := e.(MaxActiveConnectionErr)
	return ok
}
