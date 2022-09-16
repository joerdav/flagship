package models

type ThrottleConfig struct {
	// Whitelist is a list of hash results that will always be allowed through the throttle.
	Whitelist []uint `json:"whitelist,omitempty"`
	// Probability of a hash result making it through the throttle.
	Probability float64 `json:"probability,omitempty"`
}

type Features map[string]interface{}

func (f Features) Bool(s string) bool {
	b, ok := f[s].(bool)
	return b && ok
}

func (f Features) All() Features {
	return f
}

type StoreDocument struct {
	Features  Features                  `json:"features"`
	Throttles map[string]ThrottleConfig `json:"throttles"`
}
