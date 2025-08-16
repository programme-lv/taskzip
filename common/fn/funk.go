package fn

// file contains functional programming patterns
// to shorten the code and improve readability

import "sort"

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

// MinInt returns the minimum value in a non-empty int slice
func MinInt(ss []int) int {
	if len(ss) == 0 {
		return 0
	}
	min := ss[0]
	for _, v := range ss[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

// MaxInt returns the maximum value in a non-empty int slice
func MaxInt(ss []int) int {
	if len(ss) == 0 {
		return 0
	}
	max := ss[0]
	for _, v := range ss[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

// AreConsecutive reports whether ints form a contiguous ascending sequence
func AreConsecutive(ss []int) bool {
	if len(ss) == 0 {
		return false
	}
	tmp := make([]int, len(ss))
	copy(tmp, ss)
	sort.Ints(tmp)
	for i := 1; i < len(tmp); i++ {
		if tmp[i] != tmp[i-1]+1 {
			return false
		}
	}
	return true
}
