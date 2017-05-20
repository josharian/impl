package main

import (
	"fmt"
	"go/token"
	"strconv"
	"strings"
)

func getPosition(pos string) (*token.Position, error) {
	arr := strings.Split(pos, ":")

	if len(arr) < 2 {
		return nil, fmt.Errorf("Invalid position spec")
	}

	p := token.Position{Column: 1}

	p.Filename = arr[0]

	line, err := strconv.Atoi(arr[1])
	if err != nil {
		return nil, fmt.Errorf("invalid line spec in position: %s", err)
	}
	p.Line = line

	if len(arr) == 3 {
		col, err := strconv.Atoi(arr[2])
		if err != nil {
			return nil, fmt.Errorf("invalid column spec in position: %s", err)
		}
		p.Column = col
	}

	return &p, nil
}
