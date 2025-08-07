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

// Count returns the number of elements that satisfy the predicate
func Count[T any](ss []T, test func(T) bool) int {
	count := 0
	for _, s := range ss {
		if test(s) {
			count++
		}
	}
	return count
}

// Unique returns a slice with duplicate elements removed
func Unique[T comparable](ss []T) []T {
	seen := make(map[T]bool)
	var result []T
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// Any returns true if any element satisfies the predicate
func Any[T any](ss []T, test func(T) bool) bool {
	for _, s := range ss {
		if test(s) {
			return true
		}
	}
	return false
}
