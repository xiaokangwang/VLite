package reedSolomon

import "github.com/xiaokangwang/VLite/interfaces"

var defaultParitySum = interfaces.RSParityShardSum{
	ParityLookupTable: []int{
		4, 7, 8, 9, 11, 12, 13, 14, 15, 16,
		16, 17, 17, 18, 19, 19, 20, 20, 21, 21,
		22, 22, 23, 23, 24, 24, 25, 25, 26, 26,
		27, 27, 27, 27, 28, 28, 28, 28, 29, 29,
		29, 29, 30, 30, 30, 30, 31, 31, 31, 31,
		32, 32, 32, 32, 32, 32, 32, 32, 33, 33,
		33, 33, 33, 33, 33, 33, 34, 34, 34, 34,
		34, 34, 34, 34, 35, 35, 35, 35, 35, 35,
	},
}
