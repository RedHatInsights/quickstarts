package utils

// ConvertStringSlice safely converts *[]string to []string
func ConvertStringSlice(slice *[]string) []string {
	if slice == nil {
		return nil
	}
	return *slice
}

// ConvertIntPtr safely converts *int to int with default value
func ConvertIntPtr(ptr *int, defaultValue int) int {
	if ptr == nil {
		return defaultValue
	}
	return *ptr
}
