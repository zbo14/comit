package types

type Gate struct {
	NilChan
	done NilChan
}

func MakeGate() Gate {
	gate := MakeNilChan()
	done := MakeNilChan()
	go func() {
		gate.Send()
		done.Send()
	}()
	select {
	case <-done:
		return Gate{gate, done}
	}
}

func (g Gate) Enter() *struct{} {
	return <-g.NilChan
}

func (g Gate) Leave() {
	go func() {
		g.Send()
		g.done.Send()
	}()
	select {
	case <-g.done:
		return
	}
}
