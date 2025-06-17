//go:build tools

// This file is used to track tool dependencies via go modules.
// See https://github.com/golang/go/issues/25922

package tools

import (
	_ "go.uber.org/mock/mockgen"
)
