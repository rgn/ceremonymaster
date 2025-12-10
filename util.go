package main

func min(x, y int) int {
	if x > y {
		return y
	}
	return x
}

func minf(x, y float32) float32 {
	if x > y {
		return y
	}
	return x
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func maxf(x, y float32) float32 {
	if x > y {
		return x
	}
	return y
}
