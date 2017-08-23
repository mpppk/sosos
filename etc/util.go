package etc

import "strings"

func IsScript(fileName string) bool {
	scriptExtList := []string{"sh", "bat", "ps1", "rb", "py", "pl", "php"}
	for _, ext := range scriptExtList {
		if strings.Contains(fileName, "."+ext) {
			return true
		}
	}
	return false
}

func Remove(numbers []int64, search int64) []int64 {
	result := []int64{}
	for _, num := range numbers {
		if num != search {
			result = append(result, num)
		}
	}
	return result
}
