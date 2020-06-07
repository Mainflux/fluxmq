package topic

import (
	"errors"
	"sync"
)

const (
	stateCHR byte = iota // Regular character
	stateMWC             // Multi-level wildcard
	stateSWC             // Single-level wildcard
	stateSEP             // Topic level separator
	stateSYS             // System level topic ($)
)

const (
	qos0 byte = iota
	qos1
	qos2
	qosFail = 0x80
)

var (
	errMWCLast   = errors.New("multi-level wildcard not at the last level")
	errMWCEntire = errors.New("multi-level wildcard doesn't occupy entire topic level")
	errSWCEntire = errors.New("single-level wildcard doesn't occupy entire topic level")
	//errSYS       = errors.New("cannot publish to '$'")
)

type Topic struct {
	Subs map[string]Subscription
	Mu   sync.RWMutex
}

func validQos(qos byte) bool {
	return qos == qos0 || qos == qos1 || qos == qos2
}
