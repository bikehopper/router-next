package utils

func Map[T any, U any](slice []T, f func(T) U) []U {
	result := make([]U, len(slice))
	for i, item := range slice {
		result[i] = f(item)
	}

	return result
}

func Reduce[T any, U any](slice []T, init U, f func(U, T) U) U {
	acc := init
	for _, v := range slice {
		acc = f(acc, v)
	}

	return acc
}
