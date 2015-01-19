package main

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

func ConvertToUnderscore(camel string) (string, error) {
	var prevRune rune
	var underscore []rune
	for index, runeChar := range camel {
		if index == 0 {
			if !unicode.IsLetter(runeChar) {
				return "", fmt.Errorf("Table and column names can't start with a character other than a letter.")
			}
			underscore = append(underscore, unicode.ToLower(runeChar))
			prevRune = runeChar
		} else {
			if runeChar == '_' || unicode.IsLetter(runeChar) || unicode.IsDigit(runeChar) {
				//Look for Upper case letters, append _ and make character lower case
				if unicode.IsUpper(runeChar) {
					if !unicode.IsUpper(prevRune) {
						underscore = append(underscore, '_')
					}
					underscore = append(underscore, unicode.ToLower(runeChar))
				} else {
					underscore = append(underscore, runeChar)
				}
			} else {
				return "", fmt.Errorf("Table and column names can't contain non-alphanumeric characters.")
			}
		}
		prevRune = runeChar
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
