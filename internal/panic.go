// SPDX-License-Identifier: MIT
//
// Copyright (C) 2021 Daniel Bourdrez. All Rights Reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree or at
// https://spdx.org/licenses/MIT.html

package internal

import (
	"errors"
	"fmt"
)

var (
	errNoPanic        = errors.New("no panic")
	errNoPanicMessage = errors.New("panic but no message")
)

func hasPanic(f func()) (has bool, err error) {
	err = nil
	var report interface{}
	func() {
		defer func() {
			if report = recover(); report != nil {
				has = true
			}
		}()

		f()
	}()

	if has {
		err = fmt.Errorf("%v", report)
	}

	return has, err
}

// ExpectPanic executes the function f with the expectation to recover from a panic. If no panic occurred or if the
// panic message is not the one expected, ExpectPanic returns (false, error).
func ExpectPanic(expectedError error, f func()) (bool, error) {
	hasPanic, err := hasPanic(f)

	if !hasPanic {
		return false, errNoPanic
	}

	if expectedError == nil {
		return true, nil
	}

	if err == nil {
		return false, errNoPanicMessage
	}

	if err.Error() != expectedError.Error() {
		return false, fmt.Errorf("expected %q, got: %w", expectedError, err)
	}

	return true, nil
}
