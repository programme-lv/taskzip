package fn

// file contains functional programming patterns
// to shorten the code and improve readability

// Filter drops elems that don't satisfy the predicate
func Filter[T any](ss []T, test func(T) bool) (ret []T) {
	for _, s := range ss {
		if test(s) {
			ret = append(ret, s)
		}
	}
	return
}

// Map replaces each elem with the res of f on it
func Map[S any, T any](ss []S, f func(S) T) []T {
	res := make([]T, len(ss))
	for i, s := range ss {
		res[i] = f(s)
	}
	return res
}
