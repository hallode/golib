package excel

func NextColumn(col string, offset int) string {
	runes := []rune(col)
	carry := rune(offset / 26)
	remainder := rune(offset % 26)

	if remainder == 0 {
		carry--
		remainder = 26
	}

	newLastRune := runes[len(runes)-1] + remainder
	if newLastRune > 'Z' {
		newLastRune -= 26
		carry++
	}

	runes[len(runes)-1] = newLastRune
	for carry > 0 {
		if len(runes) == 1 {
			runes = append([]rune{'A'}, runes...)
		} else {
			runes[len(runes)-2]++
			if runes[len(runes)-2] > 'Z' {
				runes[len(runes)-2] -= 26
				carry++
			} else {
				carry = 0
			}
		}
		carry--
	}

	return string(runes)
}
