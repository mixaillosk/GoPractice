//Var 23 -> E, C

package main

import (
	"fmt"
	"sort"
	"time"
)

func binarySearch(a []int, target int) int {
	left, right := 0, len(a)-1

	for left <= right {
		mid := left + (right-left)/2
		if a[mid] == target {
			for mid > 0 && a[mid-1] == target {
				mid--
			}
			return mid + 1
		} else if a[mid] < target {
			left = mid + 1
		} else {
			right = mid - 1
		}
	}
	return -1
}

func taskE() {
	fmt.Println("Running task E in BS using")

	var n int
	fmt.Println("Please, input length of the 1st array: ")
	fmt.Scan(&n)

	a := make([]int, n)
	fmt.Printf("Please, input %d numbers:\n", n)
	for i := 0; i < n; i++ {
		fmt.Scan(&a[i])
	}

	var m int
	fmt.Println("Please, input length of the 2nd array: ")
	fmt.Scan(&m)

	b := make([]int, m)
	fmt.Printf("Please, input %d numbers:\n", m)
	for i := 0; i < m; i++ {
		fmt.Scan(&b[i])
	}

	for _, bj := range b {
		index := binarySearch(a, bj)
		time.Sleep(4 * time.Second / time.Duration(m))
		fmt.Printf("%d -> %d ", bj, index)
	}
}

func taskC() {
	fmt.Println("Running task C in Map using")

	var n int
	fmt.Println("Please, input length of the array:")
	fmt.Scan(&n)

	a := make([]int, n)
	fmt.Printf("Please, input %d numbers:\n", n)
	for i := 0; i < n; i++ {
		fmt.Scan(&a[i])
	}

	start := time.Now()

	freq := make(map[int]int)
	for _, num := range a {
		freq[num]++
	}

	keys := make([]int, 0, len(freq))
	for key := range freq {
		keys = append(keys, key)
	}

	sort.Ints(keys)

	maxKeep := 0
	for i := 0; i < len(keys); i++ {
		curr := freq[keys[i]]

		if i+1 < len(keys) && keys[i+1]-keys[i] == 1 {
			curr += freq[keys[i+1]]
		}

		if curr > maxKeep {
			maxKeep = curr
		}
	}

	end := time.Since(start)

	if end < time.Second {
		time.Sleep(time.Second - end)
	}

	fmt.Printf("Min elements to delete: %d", n-maxKeep)
}

func main() {

	taskE()

	taskC()
}
