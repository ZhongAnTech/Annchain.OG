package math

func MinInt(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func MaxInt(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func MaxUint64(x, y uint64) uint64 {
	if x > y {
		return x
	}
	return y
}
