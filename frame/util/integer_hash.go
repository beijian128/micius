package util

//  [https://stackoverflow.com/a/12996028]

func Hash32(x uint32) uint32 {
	x = ((x >> 16) ^ x) * 0x45d9f3b
	x = ((x >> 16) ^ x) * 0x45d9f3b
	x = (x >> 16) ^ x
	return x
}

func UnHash32(x uint32) uint32 {
	x = ((x >> 16) ^ x) * 0x119de1f3
	x = ((x >> 16) ^ x) * 0x119de1f3
	x = (x >> 16) ^ x
	return x
}

func Hash64(x uint64) uint64 {
	x = (x ^ (x >> 30)) * 0xbf58476d1ce4e5b9
	x = (x ^ (x >> 27)) * 0x94d049bb133111eb
	x = x ^ (x >> 31)
	return x
}

func UnHash64(x uint64) uint64 {
	x = (x ^ (x >> 31) ^ (x >> 62)) * 0x319642b2d24d8ec3
	x = (x ^ (x >> 27) ^ (x >> 54)) * 0x96de1b173f119089
	x = x ^ (x >> 30) ^ (x >> 60)
	return x
}
