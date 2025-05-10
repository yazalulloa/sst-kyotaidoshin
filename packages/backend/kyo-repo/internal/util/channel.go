package util

func Sum(c <-chan int64) int64 {
	var sum int64
	for i := range c {
		sum += i
	}
	return sum
}
