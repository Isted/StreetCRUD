package main

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

func ConvertToUnderscore(camel string) (string, error) {
	if camel == "" {
		return "", nil
	}

	runes := []rune(camel)
	if !unicode.IsLetter(runes[0]) {
		return "", fmt.Errorf("Table and column names can't start with a character other than a letter.")
	}

	var underscore []rune
	underscore = append(underscore, unicode.ToLower(runes[0]))

	for i := 1; i < len(runes); i++ {
		r := runes[i]

		if !(r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)) {
			return "", fmt.Errorf("Table and column names can't contain non-alphanumeric characters.")
		}

		prev := runes[i-1]

		if unicode.IsUpper(r) {
			nextIsLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
			if prev != '_' && (unicode.IsLower(prev) || (unicode.IsUpper(prev) && nextIsLower)) {
				underscore = append(underscore, '_')
			}
			underscore = append(underscore, unicode.ToLower(r))
		} else {
			underscore = append(underscore, r)
		}
	}

	return string(underscore), nil
}

func UpperCaseFirstChar(word string) string {
	runes := []rune(word)
	if len(runes) > 0 {
		if unicode.IsLower(runes[0]) {
			runes[0] = unicode.ToUpper(runes[0])
		}
	}
	return string(runes)
}

func LowerCaseFirstChar(word string) string {
	runes := []rune(word)
	if len(runes) > 0 {
		if unicode.IsUpper(runes[0]) {
			runes[0] = unicode.ToLower(runes[0])
		}
	}
	return string(runes)
}

func AddQuotesIfAnyUpperCase(dbOrSchema string) string {
	for _, letter := range dbOrSchema {
		if unicode.IsUpper(letter) {
			dbOrSchema = "\"" + dbOrSchema + "\""
			break
		}
	}
	return dbOrSchema
}

func TrimInnerSpacesToOne(spacey string) string {

	if strings.TrimSpace(spacey) == "" {
		return ""
	}
	var runeSlice []rune
	var isAtStart bool = true
	var isWord bool = false
	for _, runeChar := range spacey {
		if runeChar != ' ' && runeChar != '\t' && isAtStart {
			runeSlice = append(runeSlice, runeChar)
			isAtStart = false
			isWord = true
		} else if isWord {
			if runeChar != ' ' && runeChar != '\t' {
				runeSlice = append(runeSlice, runeChar)
			} else {
				runeSlice = append(runeSlice, ' ')
				isWord = false
			}
		} else if !isWord {
			if runeChar != ' ' && runeChar != '\t' {
				runeSlice = append(runeSlice, runeChar)
				isWord = true
			}
		}
	}
	if runeSlice[len(runeSlice)-1] == ' ' {
		return fmt.Sprint(string(runeSlice[:len(runeSlice)-1]))
	} else {
		return fmt.Sprint(string(runeSlice))
	}

}

func ChangeCaseForRange(changeMe string, startIndex int, endIndex int) string {
	if changeMe == "" || utf8.RuneCountInString(changeMe) < endIndex+1 || startIndex > endIndex || startIndex < 0 {
		return changeMe
	}
	newWord := []rune(changeMe)
	for ; startIndex <= endIndex; startIndex++ {
		newWord[startIndex] = unicode.ToLower(newWord[startIndex])
	}
	return string(newWord)
}
