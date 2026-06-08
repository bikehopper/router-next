package utils

import (
	"reflect"
	"router/pkg/types"
	"strconv"
	"strings"
)

func Map[T any, U any](slice []T, f func(T) U) []U {
	result := make([]U, len(slice))
	for i, item := range slice {
		result[i] = f(item)
	}

	return result
}

func Reduce[T any, U any](slice []T, init U, f func(U, T) U) U {
	acc := init
	for _, v := range slice {
		acc = f(acc, v)
	}

	return acc
}

func SizeOf[T any]() int {
	return int(reflect.TypeFor[T]().Size())
}

const SnapshotLineWidth = 20

func SnapshotStr[T uint32 | types.StopID](arr []T) string {
	N := len(arr)
	numRows := N / SnapshotLineWidth
	lastRowLength := N % SnapshotLineWidth

	if lastRowLength > 0 {
		numRows += 1
	}

	rows := make([]string, numRows)
	for rowIdx := range numRows {
		start := rowIdx * SnapshotLineWidth
		var end int
		if rowIdx == numRows-1 {
			end = start + lastRowLength
		} else {
			end = start + SnapshotLineWidth
		}

		row := make([]string, end-start)
		for i := start; i < end; i++ {
			row[i-start] = strconv.Itoa(int(arr[i]))
		}

		rowStr := strings.Join(row, ",")
		rows[rowIdx] = rowStr
	}
	str := strings.Join(rows, "\n")
	return str
}
