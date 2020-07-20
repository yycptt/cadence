// The MIT License (MIT)
//
// Copyright (c) 2017-2020 Uber Technologies Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
//

// Code generated by MockGen. DO NOT EDIT.
// Source: interface.go

// Package queue is a generated GoMock package.
package queue

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"

	persistence "github.com/uber/cadence/common/persistence"
	task "github.com/uber/cadence/service/history/task"
)

// MockProcessingQueueState is a mock of ProcessingQueueState interface
type MockProcessingQueueState struct {
	ctrl     *gomock.Controller
	recorder *MockProcessingQueueStateMockRecorder
}

// MockProcessingQueueStateMockRecorder is the mock recorder for MockProcessingQueueState
type MockProcessingQueueStateMockRecorder struct {
	mock *MockProcessingQueueState
}

// NewMockProcessingQueueState creates a new mock instance
func NewMockProcessingQueueState(ctrl *gomock.Controller) *MockProcessingQueueState {
	mock := &MockProcessingQueueState{ctrl: ctrl}
	mock.recorder = &MockProcessingQueueStateMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockProcessingQueueState) EXPECT() *MockProcessingQueueStateMockRecorder {
	return m.recorder
}

// Level mocks base method
func (m *MockProcessingQueueState) Level() int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Level")
	ret0, _ := ret[0].(int)
	return ret0
}

// Level indicates an expected call of Level
func (mr *MockProcessingQueueStateMockRecorder) Level() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Level", reflect.TypeOf((*MockProcessingQueueState)(nil).Level))
}

// AckLevel mocks base method
func (m *MockProcessingQueueState) AckLevel() task.Key {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AckLevel")
	ret0, _ := ret[0].(task.Key)
	return ret0
}

// AckLevel indicates an expected call of AckLevel
func (mr *MockProcessingQueueStateMockRecorder) AckLevel() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AckLevel", reflect.TypeOf((*MockProcessingQueueState)(nil).AckLevel))
}

// ReadLevel mocks base method
func (m *MockProcessingQueueState) ReadLevel() task.Key {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReadLevel")
	ret0, _ := ret[0].(task.Key)
	return ret0
}

// ReadLevel indicates an expected call of ReadLevel
func (mr *MockProcessingQueueStateMockRecorder) ReadLevel() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadLevel", reflect.TypeOf((*MockProcessingQueueState)(nil).ReadLevel))
}

// MaxLevel mocks base method
func (m *MockProcessingQueueState) MaxLevel() task.Key {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MaxLevel")
	ret0, _ := ret[0].(task.Key)
	return ret0
}

// MaxLevel indicates an expected call of MaxLevel
func (mr *MockProcessingQueueStateMockRecorder) MaxLevel() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MaxLevel", reflect.TypeOf((*MockProcessingQueueState)(nil).MaxLevel))
}

// DomainFilter mocks base method
func (m *MockProcessingQueueState) DomainFilter() DomainFilter {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DomainFilter")
	ret0, _ := ret[0].(DomainFilter)
	return ret0
}

// DomainFilter indicates an expected call of DomainFilter
func (mr *MockProcessingQueueStateMockRecorder) DomainFilter() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DomainFilter", reflect.TypeOf((*MockProcessingQueueState)(nil).DomainFilter))
}

// MockProcessingQueue is a mock of ProcessingQueue interface
type MockProcessingQueue struct {
	ctrl     *gomock.Controller
	recorder *MockProcessingQueueMockRecorder
}

// MockProcessingQueueMockRecorder is the mock recorder for MockProcessingQueue
type MockProcessingQueueMockRecorder struct {
	mock *MockProcessingQueue
}

// NewMockProcessingQueue creates a new mock instance
func NewMockProcessingQueue(ctrl *gomock.Controller) *MockProcessingQueue {
	mock := &MockProcessingQueue{ctrl: ctrl}
	mock.recorder = &MockProcessingQueueMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockProcessingQueue) EXPECT() *MockProcessingQueueMockRecorder {
	return m.recorder
}

// State mocks base method
func (m *MockProcessingQueue) State() ProcessingQueueState {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "State")
	ret0, _ := ret[0].(ProcessingQueueState)
	return ret0
}

// State indicates an expected call of State
func (mr *MockProcessingQueueMockRecorder) State() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "State", reflect.TypeOf((*MockProcessingQueue)(nil).State))
}

// Split mocks base method
func (m *MockProcessingQueue) Split(arg0 ProcessingQueueSplitPolicy) []ProcessingQueue {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Split", arg0)
	ret0, _ := ret[0].([]ProcessingQueue)
	return ret0
}

// Split indicates an expected call of Split
func (mr *MockProcessingQueueMockRecorder) Split(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Split", reflect.TypeOf((*MockProcessingQueue)(nil).Split), arg0)
}

// Merge mocks base method
func (m *MockProcessingQueue) Merge(arg0 ProcessingQueue) []ProcessingQueue {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Merge", arg0)
	ret0, _ := ret[0].([]ProcessingQueue)
	return ret0
}

// Merge indicates an expected call of Merge
func (mr *MockProcessingQueueMockRecorder) Merge(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Merge", reflect.TypeOf((*MockProcessingQueue)(nil).Merge), arg0)
}

// AddTasks mocks base method
func (m *MockProcessingQueue) AddTasks(arg0 map[task.Key]task.Task, arg1 task.Key) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "AddTasks", arg0, arg1)
}

// AddTasks indicates an expected call of AddTasks
func (mr *MockProcessingQueueMockRecorder) AddTasks(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddTasks", reflect.TypeOf((*MockProcessingQueue)(nil).AddTasks), arg0, arg1)
}

// UpdateAckLevel mocks base method
func (m *MockProcessingQueue) UpdateAckLevel() (task.Key, int) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateAckLevel")
	ret0, _ := ret[0].(task.Key)
	ret1, _ := ret[1].(int)
	return ret0, ret1
}

// UpdateAckLevel indicates an expected call of UpdateAckLevel
func (mr *MockProcessingQueueMockRecorder) UpdateAckLevel() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateAckLevel", reflect.TypeOf((*MockProcessingQueue)(nil).UpdateAckLevel))
}

// MockProcessingQueueSplitPolicy is a mock of ProcessingQueueSplitPolicy interface
type MockProcessingQueueSplitPolicy struct {
	ctrl     *gomock.Controller
	recorder *MockProcessingQueueSplitPolicyMockRecorder
}

// MockProcessingQueueSplitPolicyMockRecorder is the mock recorder for MockProcessingQueueSplitPolicy
type MockProcessingQueueSplitPolicyMockRecorder struct {
	mock *MockProcessingQueueSplitPolicy
}

// NewMockProcessingQueueSplitPolicy creates a new mock instance
func NewMockProcessingQueueSplitPolicy(ctrl *gomock.Controller) *MockProcessingQueueSplitPolicy {
	mock := &MockProcessingQueueSplitPolicy{ctrl: ctrl}
	mock.recorder = &MockProcessingQueueSplitPolicyMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockProcessingQueueSplitPolicy) EXPECT() *MockProcessingQueueSplitPolicyMockRecorder {
	return m.recorder
}

// Evaluate mocks base method
func (m *MockProcessingQueueSplitPolicy) Evaluate(arg0 ProcessingQueue) []ProcessingQueueState {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Evaluate", arg0)
	ret0, _ := ret[0].([]ProcessingQueueState)
	return ret0
}

// Evaluate indicates an expected call of Evaluate
func (mr *MockProcessingQueueSplitPolicyMockRecorder) Evaluate(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Evaluate", reflect.TypeOf((*MockProcessingQueueSplitPolicy)(nil).Evaluate), arg0)
}

// MockProcessingQueueCollection is a mock of ProcessingQueueCollection interface
type MockProcessingQueueCollection struct {
	ctrl     *gomock.Controller
	recorder *MockProcessingQueueCollectionMockRecorder
}

// MockProcessingQueueCollectionMockRecorder is the mock recorder for MockProcessingQueueCollection
type MockProcessingQueueCollectionMockRecorder struct {
	mock *MockProcessingQueueCollection
}

// NewMockProcessingQueueCollection creates a new mock instance
func NewMockProcessingQueueCollection(ctrl *gomock.Controller) *MockProcessingQueueCollection {
	mock := &MockProcessingQueueCollection{ctrl: ctrl}
	mock.recorder = &MockProcessingQueueCollectionMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockProcessingQueueCollection) EXPECT() *MockProcessingQueueCollectionMockRecorder {
	return m.recorder
}

// Level mocks base method
func (m *MockProcessingQueueCollection) Level() int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Level")
	ret0, _ := ret[0].(int)
	return ret0
}

// Level indicates an expected call of Level
func (mr *MockProcessingQueueCollectionMockRecorder) Level() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Level", reflect.TypeOf((*MockProcessingQueueCollection)(nil).Level))
}

// Queues mocks base method
func (m *MockProcessingQueueCollection) Queues() []ProcessingQueue {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Queues")
	ret0, _ := ret[0].([]ProcessingQueue)
	return ret0
}

// Queues indicates an expected call of Queues
func (mr *MockProcessingQueueCollectionMockRecorder) Queues() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Queues", reflect.TypeOf((*MockProcessingQueueCollection)(nil).Queues))
}

// ActiveQueue mocks base method
func (m *MockProcessingQueueCollection) ActiveQueue() ProcessingQueue {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ActiveQueue")
	ret0, _ := ret[0].(ProcessingQueue)
	return ret0
}

// ActiveQueue indicates an expected call of ActiveQueue
func (mr *MockProcessingQueueCollectionMockRecorder) ActiveQueue() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ActiveQueue", reflect.TypeOf((*MockProcessingQueueCollection)(nil).ActiveQueue))
}

// AddTasks mocks base method
func (m *MockProcessingQueueCollection) AddTasks(arg0 map[task.Key]task.Task, arg1 task.Key) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "AddTasks", arg0, arg1)
}

// AddTasks indicates an expected call of AddTasks
func (mr *MockProcessingQueueCollectionMockRecorder) AddTasks(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddTasks", reflect.TypeOf((*MockProcessingQueueCollection)(nil).AddTasks), arg0, arg1)
}

// UpdateAckLevels mocks base method
func (m *MockProcessingQueueCollection) UpdateAckLevels() (task.Key, int) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateAckLevels")
	ret0, _ := ret[0].(task.Key)
	ret1, _ := ret[1].(int)
	return ret0, ret1
}

// UpdateAckLevels indicates an expected call of UpdateAckLevels
func (mr *MockProcessingQueueCollectionMockRecorder) UpdateAckLevels() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateAckLevels", reflect.TypeOf((*MockProcessingQueueCollection)(nil).UpdateAckLevels))
}

// Split mocks base method
func (m *MockProcessingQueueCollection) Split(arg0 ProcessingQueueSplitPolicy) []ProcessingQueue {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Split", arg0)
	ret0, _ := ret[0].([]ProcessingQueue)
	return ret0
}

// Split indicates an expected call of Split
func (mr *MockProcessingQueueCollectionMockRecorder) Split(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Split", reflect.TypeOf((*MockProcessingQueueCollection)(nil).Split), arg0)
}

// Merge mocks base method
func (m *MockProcessingQueueCollection) Merge(arg0 []ProcessingQueue) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Merge", arg0)
}

// Merge indicates an expected call of Merge
func (mr *MockProcessingQueueCollectionMockRecorder) Merge(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Merge", reflect.TypeOf((*MockProcessingQueueCollection)(nil).Merge), arg0)
}

// MockProcessor is a mock of Processor interface
type MockProcessor struct {
	ctrl     *gomock.Controller
	recorder *MockProcessorMockRecorder
}

// MockProcessorMockRecorder is the mock recorder for MockProcessor
type MockProcessorMockRecorder struct {
	mock *MockProcessor
}

// NewMockProcessor creates a new mock instance
func NewMockProcessor(ctrl *gomock.Controller) *MockProcessor {
	mock := &MockProcessor{ctrl: ctrl}
	mock.recorder = &MockProcessorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockProcessor) EXPECT() *MockProcessorMockRecorder {
	return m.recorder
}

// Start mocks base method
func (m *MockProcessor) Start() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Start")
}

// Start indicates an expected call of Start
func (mr *MockProcessorMockRecorder) Start() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockProcessor)(nil).Start))
}

// Stop mocks base method
func (m *MockProcessor) Stop() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Stop")
}

// Stop indicates an expected call of Stop
func (mr *MockProcessorMockRecorder) Stop() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stop", reflect.TypeOf((*MockProcessor)(nil).Stop))
}

// FailoverDomain mocks base method
func (m *MockProcessor) FailoverDomain(domainIDs map[string]struct{}) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "FailoverDomain", domainIDs)
}

// FailoverDomain indicates an expected call of FailoverDomain
func (mr *MockProcessorMockRecorder) FailoverDomain(domainIDs interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FailoverDomain", reflect.TypeOf((*MockProcessor)(nil).FailoverDomain), domainIDs)
}

// NotifyNewTask mocks base method
func (m *MockProcessor) NotifyNewTask(clusterName string, transferTasks []persistence.Task) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "NotifyNewTask", clusterName, transferTasks)
}

// NotifyNewTask indicates an expected call of NotifyNewTask
func (mr *MockProcessorMockRecorder) NotifyNewTask(clusterName, transferTasks interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NotifyNewTask", reflect.TypeOf((*MockProcessor)(nil).NotifyNewTask), clusterName, transferTasks)
}

// LockTaskProcessing mocks base method
func (m *MockProcessor) LockTaskProcessing() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "LockTaskProcessing")
}

// LockTaskProcessing indicates an expected call of LockTaskProcessing
func (mr *MockProcessorMockRecorder) LockTaskProcessing() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LockTaskProcessing", reflect.TypeOf((*MockProcessor)(nil).LockTaskProcessing))
}

// UnlockTaskProcessing mocks base method
func (m *MockProcessor) UnlockTaskProcessing() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "UnlockTaskProcessing")
}

// UnlockTaskProcessing indicates an expected call of UnlockTaskProcessing
func (mr *MockProcessorMockRecorder) UnlockTaskProcessing() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UnlockTaskProcessing", reflect.TypeOf((*MockProcessor)(nil).UnlockTaskProcessing))
}
