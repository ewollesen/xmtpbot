// Copyright 2016 Eric Wollesen <ericw at xmtp dot net>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package test

import (
	"testing"

	"github.com/spacemonkeygo/errors"
)

type Test struct {
	*testing.T
}

func New(t *testing.T) *Test {
	return &Test{
		T: t,
	}
}

func (t *Test) Assert(prop bool, msg ...string) {
	if !prop {
		t.Logf("failed: %s", msg)
		t.Fail()
	}
}

func (t *Test) AssertNil(s interface{}) {
	if s != nil {
		t.Logf("expected %+v to be nil", s)
		t.Fail()
	}
}

func (t *Test) AssertEqual(actual, expected interface{}, msg ...string) {
	if actual != expected {
		t.Logf("expected %+v == %+v", expected, actual)
		t.Fail()
	}
}

func (t *Test) AssertErrorContains(err error, error_class *errors.ErrorClass) {
	if !error_class.Contains(err) {
		t.Logf("expected %+v to be %+v", err, error_class)
		t.Fail()
	}
}
