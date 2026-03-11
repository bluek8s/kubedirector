//go:build tools
// +build tools

// Package tools tracks build tool dependencies for go mod.
// See https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module
package tools

import _ "k8s.io/code-generator/cmd/deepcopy-gen"