package containers

import "fmt"

// RestartPolicyCondition is a type to describe possible restart policies
type RestartPolicyCondition string

const (
	// Any Always restarts
	Any RestartPolicyCondition = "any"
	// None Never restarts
	None RestartPolicyCondition = "none"
	// OnFailure Restarts only on failure
	OnFailure RestartPolicyCondition = "on-failure"
)

// RestartPolicyConditionValues includes all the possible values for RestartPolicyCondition
var RestartPolicyConditionValues = []RestartPolicyCondition{
	Any,
	None,
	OnFailure,
}

// String returns the string format of the human readable memory bytes
func (m *RestartPolicyCondition) String() string {
	return string(*m)
}

// Set sets the value of the MemBytes by passing a string
func (m *RestartPolicyCondition) Set(val string) error {
	found := false
	for _, v := range RestartPolicyConditionValues {
		if string(v) == val {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("unknown restart policy condition %q. Accepted values are %q", val, RestartPolicyConditionValues)
	}
	*m = RestartPolicyCondition(val)
	return nil
}

// Type returns the type
func (m *RestartPolicyCondition) Type() string {
	return "restart-policy-condition"
}

// Value returns the value in int64
func (m *RestartPolicyCondition) Value() int64 {
	for i, v := range RestartPolicyConditionValues {
		if *m == v {
			return int64(i)
		}
	}
	return -1
}
