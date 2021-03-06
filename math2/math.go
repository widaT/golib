package math2

import (
	"math"
	"errors"
)
//求欧几里距离
func Euclidean(infoA, infoB []float64) (float64,error) {
	if len(infoA) != len(infoB) {
		return 0,errors.New("param error")
	}
	var distance float64
	for i, number := range infoA {
		//distance += math.Pow(number-infoB[i], 2)
		distance += (number-infoB[i]) * (number-infoB[i]) //效率比上一个高多个数量级
	}
	return math.Sqrt(distance),nil
}

//求余弦相似度
func Cos(infoA, infoB []float64) (float64,error) {
	if len(infoA) != len(infoB) {
		return 0,errors.New("param error")
	}
	var a,b,c float64
	for i, number := range infoA {
		a += number*infoB[i]
	}
	for _, number := range infoA {
		//b += math.Pow(number, 2)
		b += number*number
	}
	b = math.Sqrt(b)
	for _, number := range infoB {
		//c += math.Pow(number, 2)
		c += number*number
	}
	c= math.Sqrt(c)
	return a/(b*c),nil
}

