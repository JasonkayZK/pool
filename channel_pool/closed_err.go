package channel_pool

/*
	A error type for invoke method on closed pool

	ClosedErr is the error resulting if the pool is closed via pool.Close()
 */
type ClosedErr struct {
	msg string
}

func (e ClosedErr) Error() string {
	return e.msg
}

func NewDefaultClosedErr() ClosedErr {
	return NewClosedErr("pool closed err")
}

func NewClosedErr(cause string) ClosedErr {
	return ClosedErr{
		msg: cause,
	}
}

func IsClosedErr(e error) bool {
	_, ok := e.(ClosedErr)
	return ok
}
