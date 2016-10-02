package types

type NilChan chan *struct{}

func MakeNilChan() NilChan {
	return make(NilChan, 1)
}

func (nc NilChan) Send() {
	nc <- nil
}
