// Copyright (c) 2017 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

// Code generated by mockery v1.0.0. DO NOT EDIT.

package provider

import archiver "github.com/uber/cadence/common/archiver"
import mock "github.com/stretchr/testify/mock"

import tag "github.com/uber/cadence/common/log/tag"

// ArchiverProviderMock is an autogenerated mock type for the ArchiverProvider type
type ArchiverProviderMock struct {
	mock.Mock
}

// GetHistoryArchiver provides a mock function with given fields: scheme, serviceName, tags
func (_m *ArchiverProviderMock) GetHistoryArchiver(scheme string, serviceName string, tags ...tag.Tag) (archiver.HistoryArchiver, error) {
	_va := make([]interface{}, len(tags))
	for _i := range tags {
		_va[_i] = tags[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, scheme, serviceName)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 archiver.HistoryArchiver
	if rf, ok := ret.Get(0).(func(string, string, ...tag.Tag) archiver.HistoryArchiver); ok {
		r0 = rf(scheme, serviceName, tags...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(archiver.HistoryArchiver)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string, ...tag.Tag) error); ok {
		r1 = rf(scheme, serviceName, tags...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetVisibilityArchiver provides a mock function with given fields: scheme, serviceName, tags
func (_m *ArchiverProviderMock) GetVisibilityArchiver(scheme string, serviceName string, tags ...tag.Tag) (archiver.VisibilityArchiver, error) {
	_va := make([]interface{}, len(tags))
	for _i := range tags {
		_va[_i] = tags[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, scheme, serviceName)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 archiver.VisibilityArchiver
	if rf, ok := ret.Get(0).(func(string, string, ...tag.Tag) archiver.VisibilityArchiver); ok {
		r0 = rf(scheme, serviceName, tags...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(archiver.VisibilityArchiver)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string, ...tag.Tag) error); ok {
		r1 = rf(scheme, serviceName, tags...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
