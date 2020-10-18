package topic

import (
	"errors"
	"fmt"
	"reflect"
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
	qosFailure = 0x80
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

// Returns topic level, remaining topic levels and any errors
func nextTopicLevel(topic []byte) ([]byte, []byte, error) {
	s := stateCHR

	for i, c := range topic {
		switch c {
		case '/':
			if s == stateMWC {
				return nil, nil, errMWCLast
			}

			if i == 0 {
				return []byte("+"), topic[i+1:], nil
			}

			return topic[:i], topic[i+1:], nil

		case '#':
			if i != 0 {
				return nil, nil, errMWCEntire
			}

			s = stateMWC

		case '+':
			if i != 0 {
				return nil, nil, errSWCEntire
			}

			s = stateSWC

		// case '$':
		// 	if i == 0 {
		// 		return nil, nil, fmt.Errorf("Cannot publish to $ topics")
		// 	}

		// 	s = stateSYS

		default:
			if s == stateMWC || s == stateSWC {
				return nil, nil, fmt.Errorf("Wildcard characters '#' and '+' must occupy entire topic level")
			}

			s = stateCHR
		}
	}

	// If we got here that means we didn't hit the separator along the way, so the
	// topic is either empty, or does not contain a separator. Either way, we return
	// the full topic
	return topic, nil, nil
}

func equal(k1, k2 interface{}) bool {
	if reflect.TypeOf(k1) != reflect.TypeOf(k2) {
		return false
	}

	if reflect.ValueOf(k1).Kind() == reflect.Func {
		return &k1 == &k2
	}

	if k1 == k2 {
		return true
	}

	switch k1 := k1.(type) {
	case string:
		return k1 == k2.(string)

	case int64:
		return k1 == k2.(int64)

	case int32:
		return k1 == k2.(int32)

	case int16:
		return k1 == k2.(int16)

	case int8:
		return k1 == k2.(int8)

	case int:
		return k1 == k2.(int)

	case float32:
		return k1 == k2.(float32)

	case float64:
		return k1 == k2.(float64)

	case uint:
		return k1 == k2.(uint)

	case uint8:
		return k1 == k2.(uint8)

	case uint16:
		return k1 == k2.(uint16)

	case uint32:
		return k1 == k2.(uint32)

	case uint64:
		return k1 == k2.(uint64)

	case uintptr:
		return k1 == k2.(uintptr)
	}

	return false
}
