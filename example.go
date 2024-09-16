package main

import (
	"errors"
	"fmt"
)

func ProcessData(data []int) ([]int, error) {
	if len(data) == 0 {
		return nil, errors.New("no data provided")
	}

	result := make([]int, len(data))
	for i, value := range data {
		var err error
		result[i], err = processSingleValue(value)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func processSingleValue(value int) (int, error) {
	switch {
	case value > 100:
		return value - 100, nil
	case value > 50:
		return value * 2, nil
	case value > 0:
		return value * 3, nil
	default:
		return 0, fmt.Errorf("invalid data value: %d", value)
	}
}
