//Var 23 -> 8

package main

import (
	"fmt"
)

// func replaceOddWithZero(arr []int) {
// 	for i := range arr {
// 		if arr[i]%2 == 1 {
// 			arr[i] = 0
// 		}
// 	}
// }

func replaceOddWithZero(arr *[5]int) {
	for i := range *arr {
		if (*arr)[i]%2 == 1 {
			(*arr)[i] = 0
		}
	}
}

func main() {
	a := [5]int{1, 2, 3, 4, 5}
	b := [5]int{1, 2, 3, 4, 5}

	fmt.Println("Массив a:", a)
	fmt.Println("Массив b до изменения:", b)

	replaceOddWithZero(&b)

	fmt.Println("Массив b после изменения:", b)

	if a == b {
		fmt.Println("Массивы a и b равны")
	} else {
		fmt.Println("Массивы a и b НЕ равны")
	}

	s := a[:]

	sumB := 0
	for _, v := range b {
		sumB += v
	}

	var subSlice []int
	var start, end int
	found := false

	for i := 0; i < len(s); i++ {
		sum := 0
		for j := i; j < len(s); j++ {
			sum += s[j]
			if sum == sumB {
				start, end = i, j+1
				subSlice = s[start:end]
				found = true
				break
			}
		}
		if found {
			break
		}
	}

	if found {
		fmt.Printf("Срез s[%d:%d], элементы: %v, сумма равна сумме элементов b: %d\n", start, end, subSlice, sumB)
	} else {
		fmt.Println("Не найден подслайс с нужной суммой.")
	}

	m := make(map[int]int)
	for i := 1; i <= 5; i++ {
		m[i] = i * i
	}

	fmt.Println("\nМапа (ключ-значение):")
	for key, value := range m {
		fmt.Printf("%d -> %d\n", key, value)
	}

	m[1] = 100
	fmt.Println("\nОбновлённая мапа:")
	for key, value := range m {
		fmt.Printf("%d -> %d\n", key, value)
	}

	str := "мы открыты"

	// strUpper := strings.Map(func(r rune) rune {
	// 	return r - 32 //unicode.ToUpper(r)
	// }, str)

	runes := []rune(str)
	for i := range runes {
		runes[i] = runes[i] - 32
	}

	strUpper := string(runes)
	fmt.Println("\nСтрока в верхнем регистре:", strUpper)

	resultStr := strUpper + "!!!"
	fmt.Println("Результат после добавления '!!!':", resultStr)
}
