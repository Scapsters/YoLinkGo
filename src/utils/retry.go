package utils

const DefaultRetries int = 3

// Retry the given function some amount of times. If no success is reached, the last given error will be returned,
// The number after the function name refers to how many output values the given function has.
func Retry1(retryCount int, f func() error) error {
	var err error
	for range retryCount {
		err = f()
		if err == nil {
			break
		}
	}
	return err
}

// Retry the given function some amount of times. If no success is reached, the last given error will be returned,
// The number after the function name refers to how many output values the given function has.
func Retry2[T any](retryCount int, f func() (T, error)) (T, error) {
	var r1 T
	var err error
	for range retryCount {
		r1, err = f()
		if err == nil {
			break
		}
	}
	return r1, err
}

// Retry the given function some amount of times. If no success is reached, the last given error will be returned,
// The number after the function name refers to how many output values the given function has.
func Retry3[T1, T2 any](retryCount int, f func() (T1, T2, error)) (T1, T2, error) {
	var r1 T1
	var r2 T2
	var err error
	for range retryCount {
		r1, r2, err = f()
		if err == nil {
			break
		}
	}
	return r1, r2, err
}
