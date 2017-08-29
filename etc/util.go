package etc

import "strings"

func IsScript(fileName string, extList []string) bool {
	for _, ext := range extList {
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
