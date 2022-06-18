package hw03frequencyanalysis

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const (
	NumberOfTopWordsToReturn = 10

	reCyrillicAndDigits = `\p{Cyrillic}\w`
)

var regexExtractWord = regexp.MustCompile(fmt.Sprintf(`[^%[1]s]*(([%[1]s]+-?)*)[^%[1]s]*`, reCyrillicAndDigits))

func extractWord(s string) string {
	matches := regexExtractWord.FindStringSubmatch(s)
	if matches == nil {
		return ""
	}
	// вот здесь решил смириться с тем, что в регулярке не смог отсекать последний символ '-', поэтому делаю это вручную
	return strings.TrimRight(strings.ToLower(matches[1]), "-")
}

func Top10(input string) []string {
	// get all words and count it occurrences
	wordNumberOccurrences := make(map[string]int)
	for _, word := range strings.Fields(input) {
		word = extractWord(word)
		if word == "" {
			continue
		}

		if _, ok := wordNumberOccurrences[word]; !ok {
			wordNumberOccurrences[word] = 1
		} else {
			wordNumberOccurrences[word]++
		}
	}

	// merge all words with the same occurrences to slice and form hashMap
	// also save in slice all unique possible occurrences and sort them
	numberOccurrencesWords := make(map[int][]string)
	var allPossibleNumbersOccurrences []int
	for word, count := range wordNumberOccurrences {
		numberOccurrencesWords[count] = append(numberOccurrencesWords[count], word)

		newCount := true
		for _, v := range allPossibleNumbersOccurrences {
			if v == count {
				newCount = false
				break
			}
		}
		if newCount {
			allPossibleNumbersOccurrences = append(allPossibleNumbersOccurrences, count)
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(allPossibleNumbersOccurrences)))

	// Going throw all possible occurrences starting with the biggest and filling result slice
	var res []string
	for _, occurrences := range allPossibleNumbersOccurrences {
		sort.Strings(numberOccurrencesWords[occurrences])
		res = append(res, numberOccurrencesWords[occurrences]...)
		if len(res) >= NumberOfTopWordsToReturn {
			return res[:NumberOfTopWordsToReturn]
		}
	}
	return res
}
