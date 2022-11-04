package models

type ThrottleConfig struct {
	// Whitelist is a list of hash results that will always be allowed through the throttle.
	Whitelist []uint `json:"whitelist,omitempty"`
	// Probability of a hash result making it through the throttle.
	Probability float64 `json:"probability,omitempty"`
	// When true will force the rejection for all the requests going through the throttle
	ForceRejectAll bool `json:"forceRejectAll,omitempty"`
}

type Features map[string]interface{}

func (f Features) Bool(s string) bool {
	b, ok := f[s].(bool)
	return b && ok
}

type StoreDocument struct {
	Features  Features                  `json:"features"`
	Throttles map[string]ThrottleConfig `json:"throttles"`
}
