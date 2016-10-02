package types

type CallContext struct {
	Caller []byte
}

func NewCallContext(caller []byte) CallContext {
	return CallContext{
		Caller: caller,
	}
}
