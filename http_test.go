package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoParameters(t *testing.T) {
	path := "/test/noparams"
	m := hasPathParams(path)
	assert.Equal(t, false, m)
}

func TestSingleParam(t *testing.T) {
	path := "/test/:message.id"
	m := hasPathParams(path)
	assert.Equal(t, m, true)
}
