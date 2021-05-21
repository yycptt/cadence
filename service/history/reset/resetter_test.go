// Copyright (c) 2020 Uber Technologies, Inc.
// Portions of the Software are attributed to Copyright (c) 2021 Temporal Technologies Inc.
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

package reset

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/collection"
	"github.com/uber/cadence/common/definition"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/log/loggerimpl"
	"github.com/uber/cadence/common/mocks"
	"github.com/uber/cadence/common/persistence"
	"github.com/uber/cadence/common/types"
	"github.com/uber/cadence/service/history/config"
	"github.com/uber/cadence/service/history/constants"
	"github.com/uber/cadence/service/history/execution"
	"github.com/uber/cadence/service/history/shard"
)

type (
	workflowResetterSuite struct {
		suite.Suite
		*require.Assertions

		controller         *gomock.Controller
		mockShard          *shard.TestContext
		mockStateRebuilder *execution.MockStateRebuilder

		mockHistoryV2Mgr *mocks.HistoryV2Manager

		logger       log.Logger
		domainID     string
		workflowID   string
		baseRunID    string
		currentRunID string
		resetRunID   string

		workflowResetter *workflowResetterImpl
	}
)

func TestWorkflowResetterSuite(t *testing.T) {
	s := new(workflowResetterSuite)
	suite.Run(t, s)
}

func (s *workflowResetterSuite) SetupSuite() {
}

func (s *workflowResetterSuite) TearDownSuite() {
}

func (s *workflowResetterSuite) SetupTest() {
	s.Assertions = require.New(s.T())

	s.logger = loggerimpl.NewLoggerForTest(s.Suite)
	s.controller = gomock.NewController(s.T())
	s.mockStateRebuilder = execution.NewMockStateRebuilder(s.controller)

	s.mockShard = shard.NewTestContext(
		s.controller,
		&persistence.ShardInfo{
			ShardID:          0,
			RangeID:          1,
			TransferAckLevel: 0,
		},
		config.NewForTest(),
	)
	s.mockHistoryV2Mgr = s.mockShard.Resource.HistoryMgr

	s.workflowResetter = NewWorkflowResetter(
		s.mockShard,
		execution.NewCache(s.mockShard),
		s.logger,
	).(*workflowResetterImpl)
	s.workflowResetter.newStateRebuilder = func() execution.StateRebuilder {
		return s.mockStateRebuilder
	}

	s.domainID = constants.TestDomainID
	s.workflowID = "some random workflow ID"
	s.baseRunID = uuid.New()
	s.currentRunID = uuid.New()
	s.resetRunID = uuid.New()
}

func (s *workflowResetterSuite) TearDownTest() {
	s.controller.Finish()
	s.mockShard.Finish(s.T())
}

func (s *workflowResetterSuite) TestPersistToDB_CurrentTerminated() {
	currentWorkflowTerminated := true

	currentWorkflow := execution.NewMockWorkflow(s.controller)
	currentReleaseCalled := false
	currentContext := execution.NewMockContext(s.controller)
	currentMutableState := execution.NewMockMutableState(s.controller)
	var currentReleaseFn execution.ReleaseFunc = func(error) { currentReleaseCalled = true }
	currentWorkflow.EXPECT().GetContext().Return(currentContext).AnyTimes()
	currentWorkflow.EXPECT().GetMutableState().Return(currentMutableState).AnyTimes()
	currentWorkflow.EXPECT().GetReleaseFn().Return(currentReleaseFn).AnyTimes()

	resetWorkflow := execution.NewMockWorkflow(s.controller)
	resetReleaseCalled := false
	resetContext := execution.NewMockContext(s.controller)
	resetMutableState := execution.NewMockMutableState(s.controller)
	var targetReleaseFn execution.ReleaseFunc = func(error) { resetReleaseCalled = true }
	resetWorkflow.EXPECT().GetContext().Return(resetContext).AnyTimes()
	resetWorkflow.EXPECT().GetMutableState().Return(resetMutableState).AnyTimes()
	resetWorkflow.EXPECT().GetReleaseFn().Return(targetReleaseFn).AnyTimes()

	currentContext.EXPECT().UpdateWorkflowExecutionWithNewAsActive(
		gomock.Any(),
		gomock.Any(),
		resetContext,
		resetMutableState,
	).Return(nil).Times(1)

	err := s.workflowResetter.persistToDB(context.Background(), currentWorkflowTerminated, currentWorkflow, resetWorkflow)
	s.NoError(err)
	// persistToDB function is not charged of releasing locks
	s.False(currentReleaseCalled)
	s.False(resetReleaseCalled)
}

func (s *workflowResetterSuite) TestPersistToDB_CurrentNotTerminated() {
	currentWorkflowTerminated := false
	currentLastWriteVersion := int64(1234)

	currentWorkflow := execution.NewMockWorkflow(s.controller)
	currentReleaseCalled := false
	currentContext := execution.NewMockContext(s.controller)
	currentMutableState := execution.NewMockMutableState(s.controller)
	var currentReleaseFn execution.ReleaseFunc = func(error) { currentReleaseCalled = true }
	currentWorkflow.EXPECT().GetContext().Return(currentContext).AnyTimes()
	currentWorkflow.EXPECT().GetMutableState().Return(currentMutableState).AnyTimes()
	currentWorkflow.EXPECT().GetReleaseFn().Return(currentReleaseFn).AnyTimes()

	currentMutableState.EXPECT().GetExecutionInfo().Return(&persistence.WorkflowExecutionInfo{
		RunID: s.currentRunID,
	}).AnyTimes()
	currentMutableState.EXPECT().GetLastWriteVersion().Return(currentLastWriteVersion, nil).AnyTimes()

	resetWorkflow := execution.NewMockWorkflow(s.controller)
	resetReleaseCalled := false
	resetContext := execution.NewMockContext(s.controller)
	resetMutableState := execution.NewMockMutableState(s.controller)
	var targetReleaseFn execution.ReleaseFunc = func(error) { resetReleaseCalled = true }
	resetWorkflow.EXPECT().GetContext().Return(resetContext).AnyTimes()
	resetWorkflow.EXPECT().GetMutableState().Return(resetMutableState).AnyTimes()
	resetWorkflow.EXPECT().GetReleaseFn().Return(targetReleaseFn).AnyTimes()

	resetSnapshot := &persistence.WorkflowSnapshot{}
	resetEventsSeq := []*persistence.WorkflowEvents{{
		DomainID:    s.domainID,
		WorkflowID:  s.workflowID,
		RunID:       s.resetRunID,
		BranchToken: []byte("some random reset branch token"),
		Events: []*types.HistoryEvent{{
			EventID: 123,
		}},
	}}
	resetEventsSize := int64(4321)
	resetMutableState.EXPECT().CloseTransactionAsSnapshot(
		gomock.Any(),
		execution.TransactionPolicyActive,
	).Return(resetSnapshot, resetEventsSeq, nil).Times(1)
	resetContext.EXPECT().PersistNonFirstWorkflowEvents(gomock.Any(), resetEventsSeq[0]).Return(resetEventsSize, nil).Times(1)
	resetContext.EXPECT().CreateWorkflowExecution(
		gomock.Any(),
		resetSnapshot,
		resetEventsSize,
		gomock.Any(),
		persistence.CreateWorkflowModeContinueAsNew,
		s.currentRunID,
		currentLastWriteVersion,
	).Return(nil).Times(1)

	err := s.workflowResetter.persistToDB(context.Background(), currentWorkflowTerminated, currentWorkflow, resetWorkflow)
	s.NoError(err)
	// persistToDB function is not charged of releasing locks
	s.False(currentReleaseCalled)
	s.False(resetReleaseCalled)
}

func (s *workflowResetterSuite) TestReplayResetWorkflow() {
	ctx := context.Background()
	baseBranchToken := []byte("some random base branch token")
	baseRebuildLastEventID := int64(1233)
	baseRebuildLastEventVersion := int64(12)
	baseNodeID := baseRebuildLastEventID + 1

	resetBranchToken := []byte("some random reset branch token")
	resetRequestID := uuid.New()
	resetHistorySize := int64(4411)
	resetMutableState := execution.NewMockMutableState(s.controller)

	s.mockHistoryV2Mgr.On("ForkHistoryBranch", mock.Anything, &persistence.ForkHistoryBranchRequest{
		ForkBranchToken: baseBranchToken,
		ForkNodeID:      baseNodeID,
		Info:            persistence.BuildHistoryGarbageCleanupInfo(s.domainID, s.workflowID, s.resetRunID),
		ShardID:         common.IntPtr(s.mockShard.GetShardID()),
	}).Return(&persistence.ForkHistoryBranchResponse{NewBranchToken: resetBranchToken}, nil).Times(1)

	s.mockStateRebuilder.EXPECT().Rebuild(
		ctx,
		gomock.Any(),
		definition.NewWorkflowIdentifier(
			s.domainID,
			s.workflowID,
			s.baseRunID,
		),
		baseBranchToken,
		baseRebuildLastEventID,
		baseRebuildLastEventVersion,
		definition.NewWorkflowIdentifier(
			s.domainID,
			s.workflowID,
			s.resetRunID,
		),
		resetBranchToken,
		resetRequestID,
	).Return(resetMutableState, resetHistorySize, nil).Times(1)

	resetWorkflow, err := s.workflowResetter.replayResetWorkflow(
		ctx,
		s.domainID,
		s.workflowID,
		s.baseRunID,
		baseBranchToken,
		baseRebuildLastEventID,
		baseRebuildLastEventVersion,
		s.resetRunID,
		resetRequestID,
	)
	s.NoError(err)
	s.Equal(resetHistorySize, resetWorkflow.GetContext().GetHistorySize())
	s.Equal(resetMutableState, resetWorkflow.GetMutableState())
}

func (s *workflowResetterSuite) TestFailInflightActivity() {
	now := time.Now().UTC()
	terminateReason := "some random termination reason"

	mutableState := execution.NewMockMutableState(s.controller)

	activity1 := &persistence.ActivityInfo{
		Version:         12,
		ScheduleID:      123,
		StartedID:       124,
		Details:         []byte("some random activity 1 details"),
		StartedIdentity: "some random activity 1 started identity",
	}
	activity2 := &persistence.ActivityInfo{
		Version:       12,
		ScheduleID:    456,
		ScheduledTime: now.Add(-10 * time.Second),
		StartedID:     common.EmptyEventID,
	}
	mutableState.EXPECT().GetPendingActivityInfos().Return(map[int64]*persistence.ActivityInfo{
		activity1.ScheduleID: activity1,
		activity2.ScheduleID: activity2,
	}).AnyTimes()

	mutableState.EXPECT().AddActivityTaskFailedEvent(
		activity1.ScheduleID,
		activity1.StartedID,
		&types.RespondActivityTaskFailedRequest{
			Reason:   common.StringPtr(terminateReason),
			Details:  activity1.Details,
			Identity: activity1.StartedIdentity,
		},
	).Return(&types.HistoryEvent{}, nil).Times(1)
	mutableState.EXPECT().UpdateActivity(&persistence.ActivityInfo{
		Version:       activity2.Version,
		ScheduleID:    activity2.ScheduleID,
		ScheduledTime: now,
		StartedID:     activity2.StartedID,
	})

	err := s.workflowResetter.failInflightActivity(mutableState, terminateReason, now)
	s.NoError(err)
}

func (s *workflowResetterSuite) TestGenerateBranchToken() {
	baseBranchToken := []byte("some random base branch token")
	baseNodeID := int64(1234)

	resetBranchToken := []byte("some random reset branch token")

	s.mockHistoryV2Mgr.On("ForkHistoryBranch", mock.Anything, &persistence.ForkHistoryBranchRequest{
		ForkBranchToken: baseBranchToken,
		ForkNodeID:      baseNodeID,
		Info:            persistence.BuildHistoryGarbageCleanupInfo(s.domainID, s.workflowID, s.resetRunID),
		ShardID:         common.IntPtr(s.mockShard.GetShardID()),
	}).Return(&persistence.ForkHistoryBranchResponse{NewBranchToken: resetBranchToken}, nil).Times(1)

	newBranchToken, err := s.workflowResetter.forkAndGenerateBranchToken(
		context.Background(), s.domainID, s.workflowID, baseBranchToken, baseNodeID, s.resetRunID,
	)
	s.NoError(err)
	s.Equal(resetBranchToken, newBranchToken)
}

func (s *workflowResetterSuite) TestTerminateWorkflow() {
	decision := &execution.DecisionInfo{
		Version:    123,
		ScheduleID: 1234,
		StartedID:  5678,
	}
	nextEventID := int64(666)
	terminateReason := "some random terminate reason"

	mutableState := execution.NewMockMutableState(s.controller)

	mutableState.EXPECT().GetNextEventID().Return(nextEventID).AnyTimes()
	mutableState.EXPECT().GetInFlightDecision().Return(decision, true).Times(1)
	mutableState.EXPECT().AddDecisionTaskFailedEvent(
		decision.ScheduleID,
		decision.StartedID,
		types.DecisionTaskFailedCauseForceCloseDecision,
		([]byte)(nil),
		execution.IdentityHistoryService,
		"",
		"",
		"",
		"",
		int64(0),
	).Return(&types.HistoryEvent{}, nil).Times(1)
	mutableState.EXPECT().FlushBufferedEvents().Return(nil).Times(1)
	mutableState.EXPECT().AddWorkflowExecutionTerminatedEvent(
		nextEventID,
		terminateReason,
		([]byte)(nil),
		execution.IdentityHistoryService,
	).Return(&types.HistoryEvent{}, nil).Times(1)

	err := s.workflowResetter.terminateWorkflow(mutableState, terminateReason)
	s.NoError(err)
}

func (s *workflowResetterSuite) TestReapplyContinueAsNewWorkflowEvents() {
	ctx := context.Background()
	baseFirstEventID := int64(124)
	baseNextEventID := int64(456)
	baseBranchToken := []byte("some random base branch token")

	newRunID := uuid.New()
	newFirstEventID := common.FirstEventID
	newNextEventID := int64(6)
	newBranchToken := []byte("some random new branch token")

	baseEvent1 := &types.HistoryEvent{
		EventID:                              124,
		EventType:                            types.EventTypeDecisionTaskScheduled.Ptr(),
		DecisionTaskScheduledEventAttributes: &types.DecisionTaskScheduledEventAttributes{},
	}
	baseEvent2 := &types.HistoryEvent{
		EventID:                            125,
		EventType:                          types.EventTypeDecisionTaskStarted.Ptr(),
		DecisionTaskStartedEventAttributes: &types.DecisionTaskStartedEventAttributes{},
	}
	baseEvent3 := &types.HistoryEvent{
		EventID:                              126,
		EventType:                            types.EventTypeDecisionTaskCompleted.Ptr(),
		DecisionTaskCompletedEventAttributes: &types.DecisionTaskCompletedEventAttributes{},
	}
	baseEvent4 := &types.HistoryEvent{
		EventID:   127,
		EventType: types.EventTypeWorkflowExecutionContinuedAsNew.Ptr(),
		WorkflowExecutionContinuedAsNewEventAttributes: &types.WorkflowExecutionContinuedAsNewEventAttributes{
			NewExecutionRunID: newRunID,
		},
	}

	newEvent1 := &types.HistoryEvent{
		EventID:                                 1,
		EventType:                               types.EventTypeWorkflowExecutionStarted.Ptr(),
		WorkflowExecutionStartedEventAttributes: &types.WorkflowExecutionStartedEventAttributes{},
	}
	newEvent2 := &types.HistoryEvent{
		EventID:                              2,
		EventType:                            types.EventTypeDecisionTaskScheduled.Ptr(),
		DecisionTaskScheduledEventAttributes: &types.DecisionTaskScheduledEventAttributes{},
	}
	newEvent3 := &types.HistoryEvent{
		EventID:                            3,
		EventType:                          types.EventTypeDecisionTaskStarted.Ptr(),
		DecisionTaskStartedEventAttributes: &types.DecisionTaskStartedEventAttributes{},
	}
	newEvent4 := &types.HistoryEvent{
		EventID:                              4,
		EventType:                            types.EventTypeDecisionTaskCompleted.Ptr(),
		DecisionTaskCompletedEventAttributes: &types.DecisionTaskCompletedEventAttributes{},
	}
	newEvent5 := &types.HistoryEvent{
		EventID:                                5,
		EventType:                              types.EventTypeWorkflowExecutionFailed.Ptr(),
		WorkflowExecutionFailedEventAttributes: &types.WorkflowExecutionFailedEventAttributes{},
	}

	baseEvents := []*types.HistoryEvent{baseEvent1, baseEvent2, baseEvent3, baseEvent4}
	s.mockHistoryV2Mgr.On("ReadHistoryBranchByBatch", mock.Anything, &persistence.ReadHistoryBranchRequest{
		BranchToken:   baseBranchToken,
		MinEventID:    baseFirstEventID,
		MaxEventID:    baseNextEventID,
		PageSize:      execution.NDCDefaultPageSize,
		NextPageToken: nil,
		ShardID:       common.IntPtr(s.mockShard.GetShardID()),
	}).Return(&persistence.ReadHistoryBranchByBatchResponse{
		History:       []*types.History{{Events: baseEvents}},
		NextPageToken: nil,
	}, nil).Once()

	newEvents := []*types.HistoryEvent{newEvent1, newEvent2, newEvent3, newEvent4, newEvent5}
	s.mockHistoryV2Mgr.On("ReadHistoryBranchByBatch", mock.Anything, &persistence.ReadHistoryBranchRequest{
		BranchToken:   newBranchToken,
		MinEventID:    newFirstEventID,
		MaxEventID:    newNextEventID,
		PageSize:      execution.NDCDefaultPageSize,
		NextPageToken: nil,
		ShardID:       common.IntPtr(s.mockShard.GetShardID()),
	}).Return(&persistence.ReadHistoryBranchByBatchResponse{
		History:       []*types.History{{Events: newEvents}},
		NextPageToken: nil,
	}, nil).Once()

	resetContext := execution.NewMockContext(s.controller)
	resetContext.EXPECT().Lock(gomock.Any()).Return(nil).Times(1)
	resetContext.EXPECT().Unlock().Times(1)
	resetMutableState := execution.NewMockMutableState(s.controller)
	resetContext.EXPECT().LoadWorkflowExecution(gomock.Any()).Return(resetMutableState, nil).Times(1)
	resetMutableState.EXPECT().GetNextEventID().Return(newNextEventID).AnyTimes()
	resetMutableState.EXPECT().GetCurrentBranchToken().Return(newBranchToken, nil).AnyTimes()
	resetContextCacheKey := definition.NewWorkflowIdentifier(s.domainID, s.workflowID, newRunID)
	_, _ = s.workflowResetter.executionCache.PutIfNotExist(resetContextCacheKey, resetContext)

	mutableState := execution.NewMockMutableState(s.controller)

	err := s.workflowResetter.reapplyResetAndContinueAsNewWorkflowEvents(
		ctx,
		mutableState,
		s.domainID,
		s.workflowID,
		s.baseRunID,
		baseBranchToken,
		baseFirstEventID,
		baseNextEventID,
	)
	s.NoError(err)
}

func (s *workflowResetterSuite) TestReapplyWorkflowEvents() {
	firstEventID := common.FirstEventID
	nextEventID := int64(6)
	branchToken := []byte("some random branch token")

	newRunID := uuid.New()
	event1 := &types.HistoryEvent{
		EventID:                                 1,
		EventType:                               types.EventTypeWorkflowExecutionStarted.Ptr(),
		WorkflowExecutionStartedEventAttributes: &types.WorkflowExecutionStartedEventAttributes{},
	}
	event2 := &types.HistoryEvent{
		EventID:                              2,
		EventType:                            types.EventTypeDecisionTaskScheduled.Ptr(),
		DecisionTaskScheduledEventAttributes: &types.DecisionTaskScheduledEventAttributes{},
	}
	event3 := &types.HistoryEvent{
		EventID:                            3,
		EventType:                          types.EventTypeDecisionTaskStarted.Ptr(),
		DecisionTaskStartedEventAttributes: &types.DecisionTaskStartedEventAttributes{},
	}
	event4 := &types.HistoryEvent{
		EventID:                              4,
		EventType:                            types.EventTypeDecisionTaskCompleted.Ptr(),
		DecisionTaskCompletedEventAttributes: &types.DecisionTaskCompletedEventAttributes{},
	}
	event5 := &types.HistoryEvent{
		EventID:   5,
		EventType: types.EventTypeWorkflowExecutionContinuedAsNew.Ptr(),
		WorkflowExecutionContinuedAsNewEventAttributes: &types.WorkflowExecutionContinuedAsNewEventAttributes{
			NewExecutionRunID: newRunID,
		},
	}
	events := []*types.HistoryEvent{event1, event2, event3, event4, event5}
	s.mockHistoryV2Mgr.On("ReadHistoryBranchByBatch", mock.Anything, &persistence.ReadHistoryBranchRequest{
		BranchToken:   branchToken,
		MinEventID:    firstEventID,
		MaxEventID:    nextEventID,
		PageSize:      execution.NDCDefaultPageSize,
		NextPageToken: nil,
		ShardID:       common.IntPtr(s.mockShard.GetShardID()),
	}).Return(&persistence.ReadHistoryBranchByBatchResponse{
		History:       []*types.History{{Events: events}},
		NextPageToken: nil,
	}, nil).Once()

	mutableState := execution.NewMockMutableState(s.controller)

	nextRunID, err := s.workflowResetter.reapplyWorkflowEvents(
		context.Background(),
		mutableState,
		firstEventID,
		nextEventID,
		branchToken,
	)
	s.NoError(err)
	s.Equal(newRunID, nextRunID)
}

func (s *workflowResetterSuite) TestClosePendingDecisionTask() {
	sourceMutableState := execution.NewMockMutableState(s.controller)
	baseRunID := uuid.New()
	newRunID := uuid.New()
	baseForkEventVerison := int64(10)
	reason := "test"
	decisionScheduleEventID := int64(2)
	decisionStartEventID := decisionScheduleEventID + 1

	// The workflow has decision schedule and decision start
	sourceMutableState.EXPECT().GetInFlightDecision().Return(&execution.DecisionInfo{
		ScheduleID: decisionScheduleEventID,
		StartedID:  decisionStartEventID,
	}, true).Times(1)
	sourceMutableState.EXPECT().GetPendingChildExecutionInfos().Return(make(map[int64]*persistence.ChildExecutionInfo)).Times(1)
	sourceMutableState.EXPECT().AddDecisionTaskFailedEvent(
		decisionScheduleEventID,
		decisionStartEventID,
		types.DecisionTaskFailedCauseResetWorkflow,
		nil,
		execution.IdentityHistoryService,
		reason,
		"",
		baseRunID,
		newRunID,
		baseForkEventVerison,
	).Return(nil, nil).Times(1)

	_, err := s.workflowResetter.closePendingDecisionTask(
		sourceMutableState,
		baseRunID,
		newRunID,
		baseForkEventVerison,
		reason,
	)
	s.NoError(err)

	// The workflow has only decision schedule
	sourceMutableState.EXPECT().GetInFlightDecision().Return(nil, false).Times(1)
	sourceMutableState.EXPECT().GetPendingDecision().Return(&execution.DecisionInfo{ScheduleID: decisionScheduleEventID}, true).Times(1)
	sourceMutableState.EXPECT().GetPendingChildExecutionInfos().Return(make(map[int64]*persistence.ChildExecutionInfo)).Times(1)
	sourceMutableState.EXPECT().AddDecisionTaskResetTimeoutEvent(
		decisionScheduleEventID,
		baseRunID,
		newRunID,
		baseForkEventVerison,
		reason,
	).Return(nil, nil).Times(1)
	_, err = s.workflowResetter.closePendingDecisionTask(
		sourceMutableState,
		baseRunID,
		newRunID,
		baseForkEventVerison,
		reason,
	)
	s.NoError(err)
}

func (s *workflowResetterSuite) TestReapplyEvents() {

	event1 := &types.HistoryEvent{
		EventID:   101,
		EventType: types.EventTypeWorkflowExecutionSignaled.Ptr(),
		WorkflowExecutionSignaledEventAttributes: &types.WorkflowExecutionSignaledEventAttributes{
			SignalName: "some random signal name",
			Input:      []byte("some random signal input"),
			Identity:   "some random signal identity",
		},
	}
	event2 := &types.HistoryEvent{
		EventID:                              102,
		EventType:                            types.EventTypeDecisionTaskScheduled.Ptr(),
		DecisionTaskScheduledEventAttributes: &types.DecisionTaskScheduledEventAttributes{},
	}
	event3 := &types.HistoryEvent{
		EventID:   103,
		EventType: types.EventTypeWorkflowExecutionSignaled.Ptr(),
		WorkflowExecutionSignaledEventAttributes: &types.WorkflowExecutionSignaledEventAttributes{
			SignalName: "another random signal name",
			Input:      []byte("another random signal input"),
			Identity:   "another random signal identity",
		},
	}
	events := []*types.HistoryEvent{event1, event2, event3}

	mutableState := execution.NewMockMutableState(s.controller)

	for _, event := range events {
		if event.GetEventType() == types.EventTypeWorkflowExecutionSignaled {
			attr := event.GetWorkflowExecutionSignaledEventAttributes()
			mutableState.EXPECT().AddWorkflowExecutionSignaled(
				attr.GetSignalName(),
				attr.GetInput(),
				attr.GetIdentity(),
			).Return(&types.HistoryEvent{}, nil).Times(1)
		}
	}

	err := s.workflowResetter.reapplyEvents(mutableState, events)
	s.NoError(err)
}

func (s *workflowResetterSuite) TestPagination() {
	firstEventID := common.FirstEventID
	nextEventID := int64(101)
	branchToken := []byte("some random branch token")

	event1 := &types.HistoryEvent{
		EventID:                                 1,
		WorkflowExecutionStartedEventAttributes: &types.WorkflowExecutionStartedEventAttributes{},
	}
	event2 := &types.HistoryEvent{
		EventID:                              2,
		DecisionTaskScheduledEventAttributes: &types.DecisionTaskScheduledEventAttributes{},
	}
	event3 := &types.HistoryEvent{
		EventID:                            3,
		DecisionTaskStartedEventAttributes: &types.DecisionTaskStartedEventAttributes{},
	}
	event4 := &types.HistoryEvent{
		EventID:                              4,
		DecisionTaskCompletedEventAttributes: &types.DecisionTaskCompletedEventAttributes{},
	}
	event5 := &types.HistoryEvent{
		EventID:                              5,
		ActivityTaskScheduledEventAttributes: &types.ActivityTaskScheduledEventAttributes{},
	}
	history1 := []*types.History{{[]*types.HistoryEvent{event1, event2, event3}}}
	history2 := []*types.History{{[]*types.HistoryEvent{event4, event5}}}
	history := append(history1, history2...)
	pageToken := []byte("some random token")

	s.mockHistoryV2Mgr.On("ReadHistoryBranchByBatch", mock.Anything, &persistence.ReadHistoryBranchRequest{
		BranchToken:   branchToken,
		MinEventID:    firstEventID,
		MaxEventID:    nextEventID,
		PageSize:      execution.NDCDefaultPageSize,
		NextPageToken: nil,
		ShardID:       common.IntPtr(s.mockShard.GetShardID()),
	}).Return(&persistence.ReadHistoryBranchByBatchResponse{
		History:       history1,
		NextPageToken: pageToken,
		Size:          12345,
	}, nil).Once()
	s.mockHistoryV2Mgr.On("ReadHistoryBranchByBatch", mock.Anything, &persistence.ReadHistoryBranchRequest{
		BranchToken:   branchToken,
		MinEventID:    firstEventID,
		MaxEventID:    nextEventID,
		PageSize:      execution.NDCDefaultPageSize,
		NextPageToken: pageToken,
		ShardID:       common.IntPtr(s.mockShard.GetShardID()),
	}).Return(&persistence.ReadHistoryBranchByBatchResponse{
		History:       history2,
		NextPageToken: nil,
		Size:          67890,
	}, nil).Once()

	paginationFn := s.workflowResetter.getPaginationFn(context.Background(), firstEventID, nextEventID, branchToken)
	iter := collection.NewPagingIterator(paginationFn)

	result := []*types.History{}
	for iter.HasNext() {
		item, err := iter.Next()
		s.NoError(err)
		result = append(result, item.(*types.History))
	}

	s.Equal(history, result)
}
