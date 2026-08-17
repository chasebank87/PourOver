//go:build !unix

package tty

import "errors"

func disableEcho() (func(), error) {
	return nil, errors.New("echo control: not supported")
}
