package errs

/*
	A error type for reach pool max connection limit
*/
type WaitConnectionTimeoutErr struct {
	msg string
}

func (e WaitConnectionTimeoutErr) Error() string {
	return e.msg
}

func NewWaitConnectionTimeoutErr(cause string) WaitConnectionTimeoutErr {
	return WaitConnectionTimeoutErr{
		msg: cause,
	}
}

func IsWaitConnectionTimeoutErr(e error) bool {
	_, ok := e.(WaitConnectionTimeoutErr)
	return ok
}
