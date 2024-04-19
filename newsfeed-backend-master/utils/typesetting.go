package utils

import "math"

const (
	ONE_LINE_WIDTH = 18 * 100.0
	// ===== internal size ======
	// English lower, one line has 37, ONE_LINE_WIDTH/37~=48.7
	ENGLISH_LOWER_WIDTH = 48.7
	// English uppwer, one line has 28, ONE_LINE_WIDTH/37~=64.3
	ENGLISH_UPPER_WIDTH = 64.3
	// numbers, one line has 32, ONE_LINE_WIDTH/37~=56.3
	NUMBER_WIDTH = 56.3
	// basic ASCII 32~47, 58~64, 91~96, 123~132, one line has 34
	BASIC_NONE_LETTER_WIDTH = 53
	// chinese charactor: 100  unit
	// others						: 100 width unit
	DEFAULT_WIDTH  = 10
	DEFAULT_SUFFIX = "..."
)

func GetOneline(text string, defaultSuffix bool) string {
	widthQuota := ONE_LINE_WIDTH
	oneline := ""
	if defaultSuffix {
		suffixWidth := CalculateWidth(DEFAULT_SUFFIX)
		widthQuota -= float64(suffixWidth)
	}

	runes := []rune(text)
	for _, rune := range runes {
		runeWidth := getOneRuneInternalSize(rune)
		if widthQuota >= runeWidth {
			oneline += string(rune)
			widthQuota -= runeWidth
		} else {
			widthQuota = -1 // use it as a flag
			break
		}
	}

	if defaultSuffix && widthQuota < 0 {
		oneline += DEFAULT_SUFFIX
	}

	return oneline
}

func CalculateWidth(text string) int {
	runes := []rune(text)
	var totalWidth = 0.0
	for _, r := range runes {
		totalWidth += getOneRuneInternalSize(r)
	}
	return int(math.Ceil(totalWidth))
}

func getOneRuneInternalSize(r rune) float64 {
	if isEnglishLetterLowercase(r) {
		return ENGLISH_LOWER_WIDTH
	}
	if isEnglishLettterUppercase(r) {
		return ENGLISH_UPPER_WIDTH
	}
	if isNumber(r) {
		return NUMBER_WIDTH
	}
	if isBasicNoneLetterChar(r) {
		return BASIC_NONE_LETTER_WIDTH
	}
	return DEFAULT_WIDTH
}

func isEnglishLetterLowercase(r rune) bool {
	if r >= rune('a') && r <= rune('z') {
		return true
	}
	return false
}

func isEnglishLettterUppercase(r rune) bool {
	if r >= rune('A') && r <= rune('Z') {
		return true
	}
	return false
}

func isNumber(r rune) bool {
	if r >= rune('0') && r <= rune('9') {
		return true
	}
	return false
}

func isBasicNoneLetterChar(r rune) bool {
	if r >= rune(' ') && r <= rune('/') { // 32~47
		return true
	}
	if r >= rune(':') && r <= rune('@') { //58~64
		return true
	}
	if r >= rune('[') && r <= rune('`') { // 91~96
		return true
	}
	if r >= rune('{') && r <= rune('~') { // 123~126
		return true
	}
	return false
}
