package types

import (
	log "github.com/inconshreveable/log15"
)

type Logger log.Logger

func NewLogger(name string) Logger {
	return log.New("module", name)
}
