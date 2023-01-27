package util

func In[T comparable](arr []T, target T) bool {
	for _, e := range arr {
		if e == target {
			return true
		}
	}
	return false
}
