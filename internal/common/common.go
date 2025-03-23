package common

func FindFirstIndex[T comparable](slice []T, item T) int {
	for index, value := range slice {
		if value == item {
			return index
		}
	}

	return -1
}
