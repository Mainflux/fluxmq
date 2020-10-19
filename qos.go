package fluxmq

const (
	QoS0 byte = iota // at most once
	QoS1             // at least once
	QoS2             // exactly once
)

// ValidQos checks the QoS value to see if it's valid. Valid QoS are QosAtMostOnce,
// QosAtLeastonce, and QosExactlyOnce.
func ValidQos(qos byte) bool {
	return qos == QoS0 || qos == QoS1 || qos == QoS2
}
