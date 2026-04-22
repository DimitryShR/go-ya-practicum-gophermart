package service

import "unicode"

// Валидирует число, представленное в виде строки, на соотвествие алгоритму Луна
func ValidLuhnNumber(number string) bool {
	if number == "" {
		return false
	}

	sum := 0
	doubleDigit := false

	for i := len(number) - 1; i >= 0; i-- {
		r := rune(number[i])
		if !unicode.IsDigit(r) {
			return false
		}

		digit := int(r - '0')
		if doubleDigit {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		doubleDigit = !doubleDigit
	}

	return sum > 0 && sum%10 == 0
}
