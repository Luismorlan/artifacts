package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var oneLineArr [8]string = [8]string{
	"【First Squawk】HUNGARY FOR FOOOO",
	"【子陵在听歌】复兴医药从2020年3月就",
	"【快讯】泰国新冠疫情小组：随着新冠",
	"`!@#$%^&*()_+{}|:><?[]|;',./\"`!@#$",
	"abcdefghijklmnopqrstuvwxyzabcdefghijk",
	"ABCDEFGHIJKLMNOPQRSTUVWXYZAB",
	"01234567890123456789012345678901",
	"泰国新冠疫情小组泰国新冠疫情小组泰国",
}
var oneLineArrExpectedWidth = [8]int{
	1891,
	1782,
	1700,
	1802,
	1802,
	1801,
	1802,
	1800,
}

var categoriesArr [5]string = [5]string{
	"abcdefghijklmnopqrstuvwxyz",
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ",
	"012345678901234567890123456789",
	"`!@#$%^&*()_+-={}|:\"<>?[]\\;',./",
	"泰国新冠疫情小组泰国新冠疫情小组泰国新冠疫情小组",
}
var categoriesArrExpectedWidth = [5]int{1267, 1672, 1689, 1643, 2400}

var textArrForTest [3]string = [3]string{
	"【子陵在听歌】复兴医药从2020年3月就会开始投入研究",
	"【First Squawk】HUNGARY FOR ANWHERER COMPANY",
	"泰国新冠疫情小组",
}

var textArrOnelineWithSuffixExpected [3]string = [3]string{
	"【子陵在听歌】复兴医药从2020年3...",
	"【First Squawk】HUNGARY FOR A...",
	"泰国新冠疫情小组",
}

func TestTypeSetting(t *testing.T) {
	t.Run("Test return default result", func(t *testing.T) {
		width := CalculateWidth("")
		require.Equal(t, 0, width)
	})

	t.Run("Test one line examples", func(t *testing.T) {
		for idx, line := range oneLineArr {
			require.Equal(t, oneLineArrExpectedWidth[idx], CalculateWidth(line))
		}
	})

	t.Run("Test categories", func(t *testing.T) {
		require.Equal(t, categoriesArrExpectedWidth[0], CalculateWidth(categoriesArr[0]))
		require.Equal(t, categoriesArrExpectedWidth[1], CalculateWidth(categoriesArr[1]))
		require.Equal(t, categoriesArrExpectedWidth[2], CalculateWidth(categoriesArr[2]))
		require.Equal(t, categoriesArrExpectedWidth[3], CalculateWidth(categoriesArr[3]))
		require.Equal(t, categoriesArrExpectedWidth[4], CalculateWidth(categoriesArr[4]))
	})

	t.Run("Test basicNoneLetterChar", func(t *testing.T) {
		for _, c := range categoriesArr[3] {
			require.Equal(t, 53, CalculateWidth(string(c)))
		}
		for _, c := range oneLineArr[3] {
			require.Equal(t, 53, CalculateWidth(string(c)))
		}
	})

	t.Run("Test one line cases", func(t *testing.T) {
		require.Equal(t, textArrOnelineWithSuffixExpected[0], GetOneline(textArrForTest[0], true))
		require.Equal(t, textArrOnelineWithSuffixExpected[1], GetOneline(textArrForTest[1], true))
		require.Equal(t, textArrOnelineWithSuffixExpected[2], GetOneline(textArrForTest[2], true))
	})
}
