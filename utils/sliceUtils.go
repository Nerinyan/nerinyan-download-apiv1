package utils

func AppendIf[T any](tf bool, slice []T, elems ...T) []T {
	if tf {
		return append(slice, elems...)
	} else {
		return slice
	}
}
