//go:build !linux && !windows

package install

import "errors"

const supported = false

var errUnsupported = errors.New("shortcuts are not supported on this system yet")

func install(Icons) (Result, error) { return Result{}, errUnsupported }
func uninstall() error              { return errUnsupported }
func installed() bool               { return false }
