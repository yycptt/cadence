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

package history

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/uber-go/tally"

	"github.com/uber/cadence/.gen/go/history"
	workflow "github.com/uber/cadence/.gen/go/shared"
	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/cache"
	"github.com/uber/cadence/common/clock"
	"github.com/uber/cadence/common/cluster"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/metrics"
	"github.com/uber/cadence/common/mocks"
	"github.com/uber/cadence/common/persistence"
	"github.com/uber/cadence/common/xdc"
	"github.com/uber/cadence/service/history/config"
	"github.com/uber/cadence/service/history/events"
	"github.com/uber/cadence/service/history/execution"
	"github.com/uber/cadence/service/history/shard"
)

type (
	timerQueueStandbyTaskExecutorSuite struct {
		suite.Suite
		*require.Assertions

		controller               *gomock.Controller
		mockShard                *shard.TestContext
		mockTxProcessor          *MocktransferQueueProcessor
		mockReplicationProcessor *MockReplicatorQueueProcessor
		mockTimerProcessor       *MocktimerQueueProcessor
		mockDomainCache          *cache.MockDomainCache
		mockClusterMetadata      *cluster.MockMetadata
		mockNDCHistoryResender   *xdc.MockNDCHistoryResender

		mockExecutionMgr        *mocks.ExecutionManager
		mockHistoryRereplicator *xdc.MockHistoryRereplicator

		logger               log.Logger
		domainID             string
		domainEntry          *cache.DomainCacheEntry
		version              int64
		clusterName          string
		now                  time.Time
		timeSource           *clock.EventTimeSource
		fetchHistoryDuration time.Duration
		discardDuration      time.Duration

		timerQueueStandbyTaskExecutor *timerQueueStandbyTaskExecutor
	}
)

func TestTimerQueueStandbyTaskExecutorSuite(t *testing.T) {
	s := new(timerQueueStandbyTaskExecutorSuite)
	suite.Run(t, s)
}

func (s *timerQueueStandbyTaskExecutorSuite) SetupSuite() {

}

func (s *timerQueueStandbyTaskExecutorSuite) SetupTest() {
	s.Assertions = require.New(s.T())

	config := config.NewForTest()
	s.domainID = testDomainID
	s.domainEntry = testGlobalDomainEntry
	s.version = s.domainEntry.GetFailoverVersion()
	s.clusterName = cluster.TestAlternativeClusterName
	s.now = time.Now()
	s.timeSource = clock.NewEventTimeSource().Update(s.now)
	s.fetchHistoryDuration = config.StandbyTaskMissingEventsResendDelay() +
		(config.StandbyTaskMissingEventsDiscardDelay()-config.StandbyTaskMissingEventsResendDelay())/2
	s.discardDuration = config.StandbyTaskMissingEventsDiscardDelay() * 2

	s.controller = gomock.NewController(s.T())
	s.mockTxProcessor = NewMocktransferQueueProcessor(s.controller)
	s.mockReplicationProcessor = NewMockReplicatorQueueProcessor(s.controller)
	s.mockTimerProcessor = NewMocktimerQueueProcessor(s.controller)
	s.mockNDCHistoryResender = xdc.NewMockNDCHistoryResender(s.controller)
	s.mockTxProcessor.EXPECT().NotifyNewTask(gomock.Any(), gomock.Any()).AnyTimes()
	s.mockReplicationProcessor.EXPECT().notifyNewTask().AnyTimes()
	s.mockTimerProcessor.EXPECT().NotifyNewTimers(gomock.Any(), gomock.Any()).AnyTimes()

	s.mockHistoryRereplicator = &xdc.MockHistoryRereplicator{}

	s.mockShard = shard.NewTestContext(
		s.controller,
		&persistence.ShardInfo{
			RangeID:          1,
			TransferAckLevel: 0,
		},
		config,
	)
	s.mockShard.SetEventsCache(events.NewCache(
		s.mockShard.GetShardID(),
		s.mockShard.GetHistoryManager(),
		s.mockShard.GetConfig(),
		s.mockShard.GetLogger(),
		s.mockShard.GetMetricsClient(),
	))
	s.mockShard.Resource.TimeSource = s.timeSource

	// ack manager will use the domain information
	s.mockDomainCache = s.mockShard.Resource.DomainCache
	s.mockExecutionMgr = s.mockShard.Resource.ExecutionMgr
	s.mockClusterMetadata = s.mockShard.Resource.ClusterMetadata
	s.mockDomainCache.EXPECT().GetDomainByID(gomock.Any()).Return(testGlobalDomainEntry, nil).AnyTimes()
	s.mockClusterMetadata.EXPECT().GetCurrentClusterName().Return(cluster.TestCurrentClusterName).AnyTimes()
	s.mockClusterMetadata.EXPECT().GetAllClusterInfo().Return(cluster.TestAllClusterInfo).AnyTimes()
	s.mockClusterMetadata.EXPECT().IsGlobalDomainEnabled().Return(true).AnyTimes()
	s.mockClusterMetadata.EXPECT().ClusterNameForFailoverVersion(s.version).Return(s.clusterName).AnyTimes()

	s.logger = s.mockShard.GetLogger()

	h := &historyEngineImpl{
		currentClusterName:   s.mockShard.GetService().GetClusterMetadata().GetCurrentClusterName(),
		shard:                s.mockShard,
		clusterMetadata:      s.mockClusterMetadata,
		executionManager:     s.mockExecutionMgr,
		executionCache:       execution.NewCache(s.mockShard),
		logger:               s.logger,
		tokenSerializer:      common.NewJSONTaskTokenSerializer(),
		metricsClient:        s.mockShard.GetMetricsClient(),
		historyEventNotifier: events.NewNotifier(s.timeSource, metrics.NewClient(tally.NoopScope, metrics.History), func(string) int { return 0 }),
		txProcessor:          s.mockTxProcessor,
		replicatorProcessor:  s.mockReplicationProcessor,
		timerProcessor:       s.mockTimerProcessor,
	}
	s.mockShard.SetEngine(h)

	s.timerQueueStandbyTaskExecutor = newTimerQueueStandbyTaskExecutor(
		s.mockShard,
		h,
		s.mockHistoryRereplicator,
		s.mockNDCHistoryResender,
		s.logger,
		s.mockShard.GetMetricsClient(),
		s.clusterName,
		config,
		// newTaskAllocator(s.mockShard),
	).(*timerQueueStandbyTaskExecutor)
}

func (s *timerQueueStandbyTaskExecutorSuite) TearDownTest() {
	s.controller.Finish()
	s.mockShard.Finish(s.T())
	s.mockHistoryRereplicator.AssertExpectations(s.T())
}

func (s *timerQueueStandbyTaskExecutorSuite) TestProcessUserTimerTimeout_Pending() {

	workflowExecution := workflow.WorkflowExecution{
		WorkflowId: common.StringPtr("some random workflow ID"),
		RunId:      common.StringPtr(uuid.New()),
	}
	workflowType := "some random workflow type"
	taskListName := "some random task list"

	mutableState := execution.NewMutableStateBuilderWithReplicationStateWithEventV2(
		s.mockShard,
		s.mockShard.GetEventsCache(),
		s.logger,
		s.version,
		workflowExecution.GetRunId(),
		testGlobalDomainEntry,
	)
	_, err := mutableState.AddWorkflowExecutionStartedEvent(
		workflowExecution,
		&history.StartWorkflowExecutionRequest{
			DomainUUID: common.StringPtr(s.domainID),
			StartRequest: &workflow.StartWorkflowExecutionRequest{
				WorkflowType:                        &workflow.WorkflowType{Name: common.StringPtr(workflowType)},
				TaskList:                            &workflow.TaskList{Name: common.StringPtr(taskListName)},
				ExecutionStartToCloseTimeoutSeconds: common.Int32Ptr(2),
				TaskStartToCloseTimeoutSeconds:      common.Int32Ptr(1),
			},
		},
	)
	s.Nil(err)

	di := addDecisionTaskScheduledEvent(mutableState)
	event := addDecisionTaskStartedEvent(mutableState, di.ScheduleID, taskListName, uuid.New())
	di.StartedID = event.GetEventId()
	event = addDecisionTaskCompletedEvent(mutableState, di.ScheduleID, di.StartedID, nil, "some random identity")

	timerID := "timer"
	timerTimeout := 2 * time.Second
	event, _ = addTimerStartedEvent(mutableState, event.GetEventId(), timerID, int64(timerTimeout.Seconds()))
	nextEventID := event.GetEventId() + 1

	timerSequence := execution.NewTimerSequence(s.timeSource, mutableState)
	mutableState.DeleteTimerTasks()
	modified, err := timerSequence.CreateNextUserTimer()
	s.NoError(err)
	s.True(modified)
	task := mutableState.GetTimerTasks()[0]
	timerTask := &persistence.TimerTaskInfo{
		Version:             s.version,
		DomainID:            s.domainID,
		WorkflowID:          workflowExecution.GetWorkflowId(),
		RunID:               workflowExecution.GetRunId(),
		TaskID:              int64(100),
		TaskType:            persistence.TaskTypeUserTimer,
		TimeoutType:         int(workflow.TimeoutTypeStartToClose),
		VisibilityTimestamp: task.(*persistence.UserTimerTask).GetVisibilityTimestamp(),
		EventID:             event.GetEventId(),
	}

	persistenceMutableState := s.createPersistenceMutableState(mutableState, event.GetEventId(), event.GetVersion())
	s.mockExecutionMgr.On("GetWorkflowExecution", mock.Anything).Return(&persistence.GetWorkflowExecutionResponse{State: persistenceMutableState}, nil)

	s.mockShard.SetCurrentTime(s.clusterName, s.now)
	err = s.timerQueueStandbyTaskExecutor.execute(timerTask, true)
	s.Equal(ErrTaskRetry, err)

	s.mockShard.SetCurrentTime(s.clusterName, s.now.Add(s.fetchHistoryDuration))
	s.mockHistoryRereplicator.On("SendMultiWorkflowHistory",
		timerTask.DomainID, timerTask.WorkflowID,
		timerTask.RunID, nextEventID,
		timerTask.RunID, common.EndEventID,
	).Return(nil).Once()
	err = s.timerQueueStandbyTaskExecutor.execute(timerTask, true)
	s.Equal(ErrTaskRetry, err)

	s.mockShard.SetCurrentTime(s.clusterName, s.now.Add(s.discardDuration))
	err = s.timerQueueStandbyTaskExecutor.execute(timerTask, true)
	s.Equal(ErrTaskDiscarded, err)
}

func (s *timerQueueStandbyTaskExecutorSuite) TestProcessUserTimerTimeout_Success() {

	workflowExecution := workflow.WorkflowExecution{
		WorkflowId: common.StringPtr("some random workflow ID"),
		RunId:      common.StringPtr(uuid.New()),
	}
	workflowType := "some random workflow type"
	taskListName := "some random task list"

	mutableState := execution.NewMutableStateBuilderWithReplicationStateWithEventV2(s.mockShard, s.mockShard.GetEventsCache(), s.logger, s.version, workflowExecution.GetRunId(), testGlobalDomainEntry)
	_, err := mutableState.AddWorkflowExecutionStartedEvent(
		workflowExecution,
		&history.StartWorkflowExecutionRequest{
			DomainUUID: common.StringPtr(s.domainID),
			StartRequest: &workflow.StartWorkflowExecutionRequest{
				WorkflowType:                        &workflow.WorkflowType{Name: common.StringPtr(workflowType)},
				TaskList:                            &workflow.TaskList{Name: common.StringPtr(taskListName)},
				ExecutionStartToCloseTimeoutSeconds: common.Int32Ptr(2),
				TaskStartToCloseTimeoutSeconds:      common.Int32Ptr(1),
			},
		},
	)
	s.Nil(err)

	di := addDecisionTaskScheduledEvent(mutableState)
	event := addDecisionTaskStartedEvent(mutableState, di.ScheduleID, taskListName, uuid.New())
	di.StartedID = event.GetEventId()
	event = addDecisionTaskCompletedEvent(mutableState, di.ScheduleID, di.StartedID, nil, "some random identity")

	timerID := "timer"
	timerTimeout := 2 * time.Second
	event, _ = addTimerStartedEvent(mutableState, event.GetEventId(), timerID, int64(timerTimeout.Seconds()))

	timerSequence := execution.NewTimerSequence(s.timeSource, mutableState)
	mutableState.DeleteTimerTasks()
	modified, err := timerSequence.CreateNextUserTimer()
	s.NoError(err)
	s.True(modified)
	task := mutableState.GetTimerTasks()[0]
	timerTask := &persistence.TimerTaskInfo{
		Version:             s.version,
		DomainID:            s.domainID,
		WorkflowID:          workflowExecution.GetWorkflowId(),
		RunID:               workflowExecution.GetRunId(),
		TaskID:              int64(100),
		TaskType:            persistence.TaskTypeUserTimer,
		TimeoutType:         int(workflow.TimeoutTypeStartToClose),
		VisibilityTimestamp: task.(*persistence.UserTimerTask).GetVisibilityTimestamp(),
		EventID:             event.GetEventId(),
	}

	event = addTimerFiredEvent(mutableState, timerID)

	persistenceMutableState := s.createPersistenceMutableState(mutableState, event.GetEventId(), event.GetVersion())
	s.mockExecutionMgr.On("GetWorkflowExecution", mock.Anything).Return(&persistence.GetWorkflowExecutionResponse{State: persistenceMutableState}, nil).Once()

	s.mockShard.SetCurrentTime(s.clusterName, s.now)
	err = s.timerQueueStandbyTaskExecutor.execute(timerTask, true)
	s.Nil(err)
}

func (s *timerQueueStandbyTaskExecutorSuite) TestProcessUserTimerTimeout_Multiple() {

	workflowExecution := workflow.WorkflowExecution{
		WorkflowId: common.StringPtr("some random workflow ID"),
		RunId:      common.StringPtr(uuid.New()),
	}
	workflowType := "some random workflow type"
	taskListName := "some random task list"

	mutableState := execution.NewMutableStateBuilderWithReplicationStateWithEventV2(s.mockShard, s.mockShard.GetEventsCache(), s.logger, s.version, workflowExecution.GetRunId(), testGlobalDomainEntry)
	_, err := mutableState.AddWorkflowExecutionStartedEvent(
		workflowExecution,
		&history.StartWorkflowExecutionRequest{
			DomainUUID: common.StringPtr(s.domainID),
			StartRequest: &workflow.StartWorkflowExecutionRequest{
				WorkflowType:                        &workflow.WorkflowType{Name: common.StringPtr(workflowType)},
				TaskList:                            &workflow.TaskList{Name: common.StringPtr(taskListName)},
				ExecutionStartToCloseTimeoutSeconds: common.Int32Ptr(2),
				TaskStartToCloseTimeoutSeconds:      common.Int32Ptr(1),
			},
		},
	)
	s.Nil(err)

	di := addDecisionTaskScheduledEvent(mutableState)
	event := addDecisionTaskStartedEvent(mutableState, di.ScheduleID, taskListName, uuid.New())
	di.StartedID = event.GetEventId()
	event = addDecisionTaskCompletedEvent(mutableState, di.ScheduleID, di.StartedID, nil, "some random identity")

	timerID1 := "timer-1"
	timerTimeout1 := 2 * time.Second
	event, _ = addTimerStartedEvent(mutableState, event.GetEventId(), timerID1, int64(timerTimeout1.Seconds()))

	timerID2 := "timer-2"
	timerTimeout2 := 50 * time.Second
	_, _ = addTimerStartedEvent(mutableState, event.GetEventId(), timerID2, int64(timerTimeout2.Seconds()))

	timerSequence := execution.NewTimerSequence(s.timeSource, mutableState)
	mutableState.DeleteTimerTasks()
	modified, err := timerSequence.CreateNextUserTimer()
	s.NoError(err)
	s.True(modified)
	task := mutableState.GetTimerTasks()[0]
	timerTask := &persistence.TimerTaskInfo{
		Version:             s.version,
		DomainID:            s.domainID,
		WorkflowID:          workflowExecution.GetWorkflowId(),
		RunID:               workflowExecution.GetRunId(),
		TaskID:              int64(100),
		TaskType:            persistence.TaskTypeUserTimer,
		TimeoutType:         int(workflow.TimeoutTypeStartToClose),
		VisibilityTimestamp: task.(*persistence.UserTimerTask).GetVisibilityTimestamp(),
		EventID:             event.GetEventId(),
	}

	event = addTimerFiredEvent(mutableState, timerID1)

	persistenceMutableState := s.createPersistenceMutableState(mutableState, event.GetEventId(), event.GetVersion())
	s.mockExecutionMgr.On("GetWorkflowExecution", mock.Anything).Return(&persistence.GetWorkflowExecutionResponse{State: persistenceMutableState}, nil).Once()

	s.mockShard.SetCurrentTime(s.clusterName, s.now)
	err = s.timerQueueStandbyTaskExecutor.execute(timerTask, true)
	s.Nil(err)
}

func (s *timerQueueStandbyTaskExecutorSuite) TestProcessActivityTimeout_Pending() {

	workflowExecution := workflow.WorkflowExecution{
		WorkflowId: common.StringPtr("some random workflow ID"),
		RunId:      common.StringPtr(uuid.New()),
	}
	workflowType := "some random workflow type"
	taskListName := "some random task list"

	mutableState := execution.NewMutableStateBuilderWithReplicationStateWithEventV2(s.mockShard, s.mockShard.GetEventsCache(), s.logger, s.version, workflowExecution.GetRunId(), testGlobalDomainEntry)
	_, err := mutableState.AddWorkflowExecutionStartedEvent(
		workflowExecution,
		&history.StartWorkflowExecutionRequest{
			DomainUUID: common.StringPtr(s.domainID),
			StartRequest: &workflow.StartWorkflowExecutionRequest{
				WorkflowType:                        &workflow.WorkflowType{Name: common.StringPtr(workflowType)},
				TaskList:                            &workflow.TaskList{Name: common.StringPtr(taskListName)},
				ExecutionStartToCloseTimeoutSeconds: common.Int32Ptr(2),
				TaskStartToCloseTimeoutSeconds:      common.Int32Ptr(1),
			},
		},
	)
	s.Nil(err)

	di := addDecisionTaskScheduledEvent(mutableState)
	event := addDecisionTaskStartedEvent(mutableState, di.ScheduleID, taskListName, uuid.New())
	di.StartedID = event.GetEventId()
	event = addDecisionTaskCompletedEvent(mutableState, di.ScheduleID, di.StartedID, nil, "some random identity")

	tasklist := "tasklist"
	activityID := "activity"
	activityType := "activity type"
	timerTimeout := 2 * time.Second
	scheduledEvent, _ := addActivityTaskScheduledEvent(mutableState, event.GetEventId(), activityID, activityType, tasklist, []byte(nil),
		int32(timerTimeout.Seconds()), int32(timerTimeout.Seconds()), int32(timerTimeout.Seconds()), int32(timerTimeout.Seconds()))
	nextEventID := scheduledEvent.GetEventId() + 1

	timerSequence := execution.NewTimerSequence(s.timeSource, mutableState)
	mutableState.DeleteTimerTasks()
	modified, err := timerSequence.CreateNextActivityTimer()
	s.NoError(err)
	s.True(modified)
	task := mutableState.GetTimerTasks()[0]
	timerTask := &persistence.TimerTaskInfo{
		Version:             s.version,
		DomainID:            s.domainID,
		WorkflowID:          workflowExecution.GetWorkflowId(),
		RunID:               workflowExecution.GetRunId(),
		TaskID:              int64(100),
		TaskType:            persistence.TaskTypeActivityTimeout,
		TimeoutType:         int(workflow.TimeoutTypeScheduleToClose),
		VisibilityTimestamp: task.(*persistence.ActivityTimeoutTask).GetVisibilityTimestamp(),
		EventID:             scheduledEvent.GetEventId(),
	}

	persistenceMutableState := s.createPersistenceMutableState(mutableState, scheduledEvent.GetEventId(), scheduledEvent.GetVersion())
	s.mockExecutionMgr.On("GetWorkflowExecution", mock.Anything).Return(&persistence.GetWorkflowExecutionResponse{State: persistenceMutableState}, nil).Once()

	s.mockShard.SetCurrentTime(s.clusterName, s.now)
	err = s.timerQueueStandbyTaskExecutor.execute(timerTask, true)
	s.Equal(ErrTaskRetry, err)

	s.mockShard.SetCurrentTime(s.clusterName, s.now.Add(s.fetchHistoryDuration))
	s.mockHistoryRereplicator.On("SendMultiWorkflowHistory",
		timerTask.DomainID, timerTask.WorkflowID,
		timerTask.RunID, nextEventID,
		timerTask.RunID, common.EndEventID,
	).Return(nil).Once()
	err = s.timerQueueStandbyTaskExecutor.execute(timerTask, true)
	s.Equal(ErrTaskRetry, err)

	s.mockShard.SetCurrentTime(s.clusterName, s.now.Add(s.discardDuration))
	err = s.timerQueueStandbyTaskExecutor.execute(timerTask, true)
	s.Equal(ErrTaskDiscarded, err)
}

func (s *timerQueueStandbyTaskExecutorSuite) TestProcessActivityTimeout_Success() {

	workflowExecution := workflow.WorkflowExecution{
		WorkflowId: common.StringPtr("some random workflow ID"),
		RunId:      common.StringPtr(uuid.New()),
	}
	workflowType := "some random workflow type"
	taskListName := "some random task list"

	mutableState := execution.NewMutableStateBuilderWithReplicationStateWithEventV2(s.mockShard, s.mockShard.GetEventsCache(), s.logger, s.version, workflowExecution.GetRunId(), testGlobalDomainEntry)
	_, err := mutableState.AddWorkflowExecutionStartedEvent(
		workflowExecution,
		&history.StartWorkflowExecutionRequest{
			DomainUUID: common.StringPtr(s.domainID),
			StartRequest: &workflow.StartWorkflowExecutionRequest{
				WorkflowType:                        &workflow.WorkflowType{Name: common.StringPtr(workflowType)},
				TaskList:                            &workflow.TaskList{Name: common.StringPtr(taskListName)},
				ExecutionStartToCloseTimeoutSeconds: common.Int32Ptr(2),
				TaskStartToCloseTimeoutSeconds:      common.Int32Ptr(1),
			},
		},
	)
	s.Nil(err)

	di := addDecisionTaskScheduledEvent(mutableState)
	event := addDecisionTaskStartedEvent(mutableState, di.ScheduleID, taskListName, uuid.New())
	di.StartedID = event.GetEventId()
	event = addDecisionTaskCompletedEvent(mutableState, di.ScheduleID, di.StartedID, nil, "some random identity")

	identity := "identity"
	tasklist := "tasklist"
	activityID := "activity"
	activityType := "activity type"
	timerTimeout := 2 * time.Second
	scheduledEvent, _ := addActivityTaskScheduledEvent(mutableState, event.GetEventId(), activityID, activityType, tasklist, []byte(nil),
		int32(timerTimeout.Seconds()), int32(timerTimeout.Seconds()), int32(timerTimeout.Seconds()), int32(timerTimeout.Seconds()))
	startedEvent := addActivityTaskStartedEvent(mutableState, scheduledEvent.GetEventId(), identity)

	timerSequence := execution.NewTimerSequence(s.timeSource, mutableState)
	mutableState.DeleteTimerTasks()
	modified, err := timerSequence.CreateNextActivityTimer()
	s.NoError(err)
	s.True(modified)
	task := mutableState.GetTimerTasks()[0]
	timerTask := &persistence.TimerTaskInfo{
		Version:             s.version,
		DomainID:            s.domainID,
		WorkflowID:          workflowExecution.GetWorkflowId(),
		RunID:               workflowExecution.GetRunId(),
		TaskID:              int64(100),
		TaskType:            persistence.TaskTypeActivityTimeout,
		TimeoutType:         int(workflow.TimeoutTypeScheduleToClose),
		VisibilityTimestamp: task.(*persistence.ActivityTimeoutTask).GetVisibilityTimestamp(),
		EventID:             scheduledEvent.GetEventId(),
	}

	completeEvent := addActivityTaskCompletedEvent(mutableState, scheduledEvent.GetEventId(), startedEvent.GetEventId(), []byte(nil), identity)

	persistenceMutableState := s.createPersistenceMutableState(mutableState, completeEvent.GetEventId(), completeEvent.GetVersion())
	s.mockExecutionMgr.On("GetWorkflowExecution", mock.Anything).Return(&persistence.GetWorkflowExecutionResponse{State: persistenceMutableState}, nil).Once()

	s.mockShard.SetCurrentTime(s.clusterName, s.now)
	err = s.timerQueueStandbyTaskExecutor.execute(timerTask, true)
	s.Nil(err)
}

func (s *timerQueueStandbyTaskExecutorSuite) TestProcessActivityTimeout_Heartbeat_Noop() {
	workflowExecution := workflow.WorkflowExecution{
		WorkflowId: common.StringPtr("some random workflow ID"),
		RunId:      common.StringPtr(uuid.New()),
	}
	workflowType := "some random workflow type"
	taskListName := "some random task list"

	mutableState := execution.NewMutableStateBuilderWithReplicationStateWithEventV2(s.mockShard, s.mockShard.GetEventsCache(), s.logger, s.version, workflowExecution.GetRunId(), testGlobalDomainEntry)
	_, err := mutableState.AddWorkflowExecutionStartedEvent(
		workflowExecution,
		&history.StartWorkflowExecutionRequest{
			DomainUUID: common.StringPtr(s.domainID),
			StartRequest: &workflow.StartWorkflowExecutionRequest{
				WorkflowType:                        &workflow.WorkflowType{Name: common.StringPtr(workflowType)},
				TaskList:                            &workflow.TaskList{Name: common.StringPtr(taskListName)},
				ExecutionStartToCloseTimeoutSeconds: common.Int32Ptr(2),
				TaskStartToCloseTimeoutSeconds:      common.Int32Ptr(1),
			},
		},
	)
	s.Nil(err)

	di := addDecisionTaskScheduledEvent(mutableState)
	event := addDecisionTaskStartedEvent(mutableState, di.ScheduleID, taskListName, uuid.New())
	di.StartedID = event.GetEventId()
	event = addDecisionTaskCompletedEvent(mutableState, di.ScheduleID, di.StartedID, nil, "some random identity")

	identity := "identity"
	tasklist := "tasklist"
	activityID := "activity"
	activityType := "activity type"
	timerTimeout := 2 * time.Second
	heartbeatTimerTimeout := time.Second
	scheduledEvent, _ := addActivityTaskScheduledEvent(mutableState, event.GetEventId(), activityID, activityType, tasklist, []byte(nil),
		int32(timerTimeout.Seconds()), int32(timerTimeout.Seconds()), int32(timerTimeout.Seconds()), int32(heartbeatTimerTimeout.Seconds()))
	startedEvent := addActivityTaskStartedEvent(mutableState, scheduledEvent.GetEventId(), identity)

	timerSequence := execution.NewTimerSequence(s.timeSource, mutableState)
	mutableState.DeleteTimerTasks()
	modified, err := timerSequence.CreateNextActivityTimer()
	s.NoError(err)
	s.True(modified)
	task := mutableState.GetTimerTasks()[0]
	s.Equal(int(execution.TimerTypeHeartbeat), task.(*persistence.ActivityTimeoutTask).TimeoutType)
	timerTask := &persistence.TimerTaskInfo{
		Version:             s.version,
		DomainID:            s.domainID,
		WorkflowID:          workflowExecution.GetWorkflowId(),
		RunID:               workflowExecution.GetRunId(),
		TaskID:              int64(100),
		TaskType:            persistence.TaskTypeActivityTimeout,
		TimeoutType:         int(workflow.TimeoutTypeHeartbeat),
		VisibilityTimestamp: task.(*persistence.ActivityTimeoutTask).GetVisibilityTimestamp().Add(-time.Second),
		EventID:             scheduledEvent.GetEventId(),
	}

	persistenceMutableState := s.createPersistenceMutableState(mutableState, startedEvent.GetEventId(), startedEvent.GetVersion())
	s.mockExecutionMgr.On("GetWorkflowExecution", mock.Anything).Return(&persistence.GetWorkflowExecutionResponse{State: persistenceMutableState}, nil).Once()

	s.mockShard.SetCurrentTime(s.clusterName, s.now)
	err = s.timerQueueStandbyTaskExecutor.execute(timerTask, true)
	s.Nil(err)
}

func (s *timerQueueStandbyTaskExecutorSuite) TestProcessActivityTimeout_Multiple_CanUpdate() {

	workflowExecution := workflow.WorkflowExecution{
		WorkflowId: common.StringPtr("some random workflow ID"),
		RunId:      common.StringPtr(uuid.New()),
	}
	workflowType := "some random workflow type"
	taskListName := "some random task list"

	mutableState := execution.NewMutableStateBuilderWithReplicationStateWithEventV2(s.mockShard, s.mockShard.GetEventsCache(), s.logger, s.version, workflowExecution.GetRunId(), testGlobalDomainEntry)
	_, err := mutableState.AddWorkflowExecutionStartedEvent(
		workflowExecution,
		&history.StartWorkflowExecutionRequest{
			DomainUUID: common.StringPtr(s.domainID),
			StartRequest: &workflow.StartWorkflowExecutionRequest{
				WorkflowType:                        &workflow.WorkflowType{Name: common.StringPtr(workflowType)},
				TaskList:                            &workflow.TaskList{Name: common.StringPtr(taskListName)},
				ExecutionStartToCloseTimeoutSeconds: common.Int32Ptr(2),
				TaskStartToCloseTimeoutSeconds:      common.Int32Ptr(1),
			},
		},
	)
	s.Nil(err)

	di := addDecisionTaskScheduledEvent(mutableState)
	event := addDecisionTaskStartedEvent(mutableState, di.ScheduleID, taskListName, uuid.New())
	di.StartedID = event.GetEventId()
	event = addDecisionTaskCompletedEvent(mutableState, di.ScheduleID, di.StartedID, nil, "some random identity")

	identity := "identity"
	tasklist := "tasklist"
	activityID1 := "activity 1"
	activityType1 := "activity type 1"
	timerTimeout1 := 2 * time.Second
	scheduledEvent1, _ := addActivityTaskScheduledEvent(mutableState, event.GetEventId(), activityID1, activityType1, tasklist, []byte(nil),
		int32(timerTimeout1.Seconds()), int32(timerTimeout1.Seconds()), int32(timerTimeout1.Seconds()), int32(timerTimeout1.Seconds()))
	startedEvent1 := addActivityTaskStartedEvent(mutableState, scheduledEvent1.GetEventId(), identity)

	activityID2 := "activity 2"
	activityType2 := "activity type 2"
	timerTimeout2 := 20 * time.Second
	scheduledEvent2, _ := addActivityTaskScheduledEvent(mutableState, event.GetEventId(), activityID2, activityType2, tasklist, []byte(nil),
		int32(timerTimeout2.Seconds()), int32(timerTimeout2.Seconds()), int32(timerTimeout2.Seconds()), int32(timerTimeout2.Seconds()))
	addActivityTaskStartedEvent(mutableState, scheduledEvent2.GetEventId(), identity)
	activityInfo2 := mutableState.GetPendingActivityInfos()[scheduledEvent2.GetEventId()]
	activityInfo2.TimerTaskStatus |= execution.TimerTaskStatusCreatedHeartbeat
	activityInfo2.LastHeartBeatUpdatedTime = time.Now()

	timerSequence := execution.NewTimerSequence(s.timeSource, mutableState)
	mutableState.DeleteTimerTasks()
	modified, err := timerSequence.CreateNextActivityTimer()
	s.NoError(err)
	s.True(modified)
	timerTask := &persistence.TimerTaskInfo{
		Version:             s.version,
		DomainID:            s.domainID,
		WorkflowID:          workflowExecution.GetWorkflowId(),
		RunID:               workflowExecution.GetRunId(),
		TaskID:              int64(100),
		TaskType:            persistence.TaskTypeActivityTimeout,
		TimeoutType:         int(workflow.TimeoutTypeHeartbeat),
		VisibilityTimestamp: activityInfo2.LastHeartBeatUpdatedTime.Add(-5 * time.Second),
		EventID:             scheduledEvent2.GetEventId(),
	}

	completeEvent1 := addActivityTaskCompletedEvent(mutableState, scheduledEvent1.GetEventId(), startedEvent1.GetEventId(), []byte(nil), identity)

	persistenceMutableState := s.createPersistenceMutableState(mutableState, completeEvent1.GetEventId(), completeEvent1.GetVersion())
	s.mockExecutionMgr.On("GetWorkflowExecution", mock.Anything).Return(&persistence.GetWorkflowExecutionResponse{State: persistenceMutableState}, nil).Once()
	s.mockExecutionMgr.On("UpdateWorkflowExecution", mock.MatchedBy(func(input *persistence.UpdateWorkflowExecutionRequest) bool {
		s.Equal(1, len(input.UpdateWorkflowMutation.TimerTasks))
		s.Equal(1, len(input.UpdateWorkflowMutation.UpsertActivityInfos))
		mutableState.GetExecutionInfo().LastUpdatedTimestamp = input.UpdateWorkflowMutation.ExecutionInfo.LastUpdatedTimestamp
		input.RangeID = 0
		input.UpdateWorkflowMutation.ExecutionInfo.LastEventTaskID = 0
		mutableState.GetExecutionInfo().LastEventTaskID = 0
		mutableState.GetExecutionInfo().DecisionOriginalScheduledTimestamp = input.UpdateWorkflowMutation.ExecutionInfo.DecisionOriginalScheduledTimestamp
		s.Equal(&persistence.UpdateWorkflowExecutionRequest{
			UpdateWorkflowMutation: persistence.WorkflowMutation{
				ExecutionInfo:             mutableState.GetExecutionInfo(),
				ExecutionStats:            &persistence.ExecutionStats{},
				ReplicationState:          mutableState.GetReplicationState(),
				TransferTasks:             nil,
				ReplicationTasks:          nil,
				TimerTasks:                input.UpdateWorkflowMutation.TimerTasks,
				Condition:                 mutableState.GetNextEventID(),
				UpsertActivityInfos:       input.UpdateWorkflowMutation.UpsertActivityInfos,
				DeleteActivityInfos:       []int64{},
				UpsertTimerInfos:          []*persistence.TimerInfo{},
				DeleteTimerInfos:          []string{},
				UpsertChildExecutionInfos: []*persistence.ChildExecutionInfo{},
				DeleteChildExecutionInfo:  nil,
				UpsertRequestCancelInfos:  []*persistence.RequestCancelInfo{},
				DeleteRequestCancelInfo:   nil,
				UpsertSignalInfos:         []*persistence.SignalInfo{},
				DeleteSignalInfo:          nil,
				UpsertSignalRequestedIDs:  []string{},
				DeleteSignalRequestedID:   "",
				NewBufferedEvents:         nil,
				ClearBufferedEvents:       false,
			},
			NewWorkflowSnapshot: nil,
			Encoding:            common.EncodingType(s.mockShard.GetConfig().EventEncodingType(s.domainID)),
		}, input)
		return true
	})).Return(&persistence.UpdateWorkflowExecutionResponse{MutableStateUpdateSessionStats: &persistence.MutableStateUpdateSessionStats{}}, nil).Once()

	s.mockShard.SetCurrentTime(s.clusterName, s.now)
	err = s.timerQueueStandbyTaskExecutor.execute(timerTask, true)
	s.Nil(err)
}

func (s *timerQueueStandbyTaskExecutorSuite) TestProcessDecisionTimeout_Pending() {

	workflowExecution := workflow.WorkflowExecution{
		WorkflowId: common.StringPtr("some random workflow ID"),
		RunId:      common.StringPtr(uuid.New()),
	}
	workflowType := "some random workflow type"
	taskListName := "some random task list"

	mutableState := execution.NewMutableStateBuilderWithReplicationStateWithEventV2(s.mockShard, s.mockShard.GetEventsCache(), s.logger, s.version, workflowExecution.GetRunId(), testGlobalDomainEntry)
	_, err := mutableState.AddWorkflowExecutionStartedEvent(
		workflowExecution,
		&history.StartWorkflowExecutionRequest{
			DomainUUID: common.StringPtr(s.domainID),
			StartRequest: &workflow.StartWorkflowExecutionRequest{
				WorkflowType:                        &workflow.WorkflowType{Name: common.StringPtr(workflowType)},
				TaskList:                            &workflow.TaskList{Name: common.StringPtr(taskListName)},
				ExecutionStartToCloseTimeoutSeconds: common.Int32Ptr(2),
				TaskStartToCloseTimeoutSeconds:      common.Int32Ptr(1),
			},
		},
	)
	s.Nil(err)

	di := addDecisionTaskScheduledEvent(mutableState)
	startedEvent := addDecisionTaskStartedEvent(mutableState, di.ScheduleID, taskListName, uuid.New())
	nextEventID := startedEvent.GetEventId() + 1

	timerTask := &persistence.TimerTaskInfo{
		Version:             s.version,
		DomainID:            s.domainID,
		WorkflowID:          workflowExecution.GetWorkflowId(),
		RunID:               workflowExecution.GetRunId(),
		TaskID:              int64(100),
		TaskType:            persistence.TaskTypeDecisionTimeout,
		TimeoutType:         int(workflow.TimeoutTypeStartToClose),
		VisibilityTimestamp: s.now,
		EventID:             di.ScheduleID,
	}

	persistenceMutableState := s.createPersistenceMutableState(mutableState, startedEvent.GetEventId(), startedEvent.GetVersion())
	s.mockExecutionMgr.On("GetWorkflowExecution", mock.Anything).Return(&persistence.GetWorkflowExecutionResponse{State: persistenceMutableState}, nil).Once()

	s.mockShard.SetCurrentTime(s.clusterName, s.now)
	err = s.timerQueueStandbyTaskExecutor.execute(timerTask, true)
	s.Equal(ErrTaskRetry, err)

	s.mockShard.SetCurrentTime(s.clusterName, s.now.Add(s.fetchHistoryDuration))
	s.mockHistoryRereplicator.On("SendMultiWorkflowHistory",
		timerTask.DomainID, timerTask.WorkflowID,
		timerTask.RunID, nextEventID,
		timerTask.RunID, common.EndEventID,
	).Return(nil).Once()
	err = s.timerQueueStandbyTaskExecutor.execute(timerTask, true)
	s.Equal(ErrTaskRetry, err)

	s.mockShard.SetCurrentTime(s.clusterName, s.now.Add(s.discardDuration))
	err = s.timerQueueStandbyTaskExecutor.execute(timerTask, true)
	s.Equal(ErrTaskDiscarded, err)
}

func (s *timerQueueStandbyTaskExecutorSuite) TestProcessDecisionTimeout_ScheduleToStartTimer() {

	execution := workflow.WorkflowExecution{
		WorkflowId: common.StringPtr("some random workflow ID"),
		RunId:      common.StringPtr(uuid.New()),
	}

	decisionScheduleID := int64(16384)

	timerTask := &persistence.TimerTaskInfo{
		Version:             s.version,
		DomainID:            s.domainID,
		WorkflowID:          execution.GetWorkflowId(),
		RunID:               execution.GetRunId(),
		TaskID:              int64(100),
		TaskType:            persistence.TaskTypeDecisionTimeout,
		TimeoutType:         int(workflow.TimeoutTypeScheduleToStart),
		VisibilityTimestamp: s.now,
		EventID:             decisionScheduleID,
	}

	s.mockShard.SetCurrentTime(s.clusterName, s.now)
	err := s.timerQueueStandbyTaskExecutor.execute(timerTask, true)
	s.Equal(nil, err)
}

func (s *timerQueueStandbyTaskExecutorSuite) TestProcessDecisionTimeout_Success() {

	workflowExecution := workflow.WorkflowExecution{
		WorkflowId: common.StringPtr("some random workflow ID"),
		RunId:      common.StringPtr(uuid.New()),
	}
	workflowType := "some random workflow type"
	taskListName := "some random task list"

	mutableState := execution.NewMutableStateBuilderWithReplicationStateWithEventV2(s.mockShard, s.mockShard.GetEventsCache(), s.logger, s.version, workflowExecution.GetRunId(), testGlobalDomainEntry)
	_, err := mutableState.AddWorkflowExecutionStartedEvent(
		workflowExecution,
		&history.StartWorkflowExecutionRequest{
			DomainUUID: common.StringPtr(s.domainID),
			StartRequest: &workflow.StartWorkflowExecutionRequest{
				WorkflowType:                        &workflow.WorkflowType{Name: common.StringPtr(workflowType)},
				TaskList:                            &workflow.TaskList{Name: common.StringPtr(taskListName)},
				ExecutionStartToCloseTimeoutSeconds: common.Int32Ptr(2),
				TaskStartToCloseTimeoutSeconds:      common.Int32Ptr(1),
			},
		},
	)
	s.Nil(err)

	di := addDecisionTaskScheduledEvent(mutableState)
	event := addDecisionTaskStartedEvent(mutableState, di.ScheduleID, taskListName, uuid.New())
	di.StartedID = event.GetEventId()
	event = addDecisionTaskCompletedEvent(mutableState, di.ScheduleID, di.StartedID, nil, "some random identity")

	timerTask := &persistence.TimerTaskInfo{
		Version:             s.version,
		DomainID:            s.domainID,
		WorkflowID:          workflowExecution.GetWorkflowId(),
		RunID:               workflowExecution.GetRunId(),
		TaskID:              int64(100),
		TaskType:            persistence.TaskTypeDecisionTimeout,
		TimeoutType:         int(workflow.TimeoutTypeStartToClose),
		VisibilityTimestamp: s.now,
		EventID:             di.ScheduleID,
	}

	persistenceMutableState := s.createPersistenceMutableState(mutableState, event.GetEventId(), event.GetVersion())
	s.mockExecutionMgr.On("GetWorkflowExecution", mock.Anything).Return(&persistence.GetWorkflowExecutionResponse{State: persistenceMutableState}, nil).Once()

	s.mockShard.SetCurrentTime(s.clusterName, s.now)
	err = s.timerQueueStandbyTaskExecutor.execute(timerTask, true)
	s.Nil(err)
}

func (s *timerQueueStandbyTaskExecutorSuite) TestProcessWorkflowBackoffTimer_Pending() {

	workflowExecution := workflow.WorkflowExecution{
		WorkflowId: common.StringPtr("some random workflow ID"),
		RunId:      common.StringPtr(uuid.New()),
	}
	workflowType := "some random workflow type"
	taskListName := "some random task list"

	mutableState := execution.NewMutableStateBuilderWithReplicationStateWithEventV2(s.mockShard, s.mockShard.GetEventsCache(), s.logger, s.version, workflowExecution.GetRunId(), testGlobalDomainEntry)
	event, err := mutableState.AddWorkflowExecutionStartedEvent(
		workflowExecution,
		&history.StartWorkflowExecutionRequest{
			DomainUUID: common.StringPtr(s.domainID),
			StartRequest: &workflow.StartWorkflowExecutionRequest{
				WorkflowType:                        &workflow.WorkflowType{Name: common.StringPtr(workflowType)},
				TaskList:                            &workflow.TaskList{Name: common.StringPtr(taskListName)},
				ExecutionStartToCloseTimeoutSeconds: common.Int32Ptr(2),
				TaskStartToCloseTimeoutSeconds:      common.Int32Ptr(1),
			},
		},
	)
	s.Nil(err)
	nextEventID := event.GetEventId() + 1

	timerTask := &persistence.TimerTaskInfo{
		Version:             s.version,
		DomainID:            s.domainID,
		WorkflowID:          workflowExecution.GetWorkflowId(),
		RunID:               workflowExecution.GetRunId(),
		TaskID:              int64(100),
		TaskType:            persistence.TaskTypeWorkflowBackoffTimer,
		VisibilityTimestamp: s.now,
	}

	persistenceMutableState := s.createPersistenceMutableState(mutableState, event.GetEventId(), event.GetVersion())
	s.mockExecutionMgr.On("GetWorkflowExecution", mock.Anything).Return(&persistence.GetWorkflowExecutionResponse{State: persistenceMutableState}, nil).Once()

	s.mockShard.SetCurrentTime(s.clusterName, s.now)
	err = s.timerQueueStandbyTaskExecutor.execute(timerTask, true)
	s.Equal(ErrTaskRetry, err)

	s.mockShard.SetCurrentTime(s.clusterName, time.Now().Add(s.fetchHistoryDuration))
	s.mockHistoryRereplicator.On("SendMultiWorkflowHistory",
		timerTask.DomainID, timerTask.WorkflowID,
		timerTask.RunID, nextEventID,
		timerTask.RunID, common.EndEventID,
	).Return(nil).Once()
	err = s.timerQueueStandbyTaskExecutor.execute(timerTask, true)
	s.Equal(ErrTaskRetry, err)

	s.mockShard.SetCurrentTime(s.clusterName, time.Now().Add(s.discardDuration))
	err = s.timerQueueStandbyTaskExecutor.execute(timerTask, true)
	s.Equal(ErrTaskDiscarded, err)
}

func (s *timerQueueStandbyTaskExecutorSuite) TestProcessWorkflowBackoffTimer_Success() {

	workflowExecution := workflow.WorkflowExecution{
		WorkflowId: common.StringPtr("some random workflow ID"),
		RunId:      common.StringPtr(uuid.New()),
	}
	workflowType := "some random workflow type"
	taskListName := "some random task list"

	mutableState := execution.NewMutableStateBuilderWithReplicationStateWithEventV2(s.mockShard, s.mockShard.GetEventsCache(), s.logger, s.version, workflowExecution.GetRunId(), testGlobalDomainEntry)
	_, err := mutableState.AddWorkflowExecutionStartedEvent(
		workflowExecution,
		&history.StartWorkflowExecutionRequest{
			DomainUUID: common.StringPtr(s.domainID),
			StartRequest: &workflow.StartWorkflowExecutionRequest{
				WorkflowType:                        &workflow.WorkflowType{Name: common.StringPtr(workflowType)},
				TaskList:                            &workflow.TaskList{Name: common.StringPtr(taskListName)},
				ExecutionStartToCloseTimeoutSeconds: common.Int32Ptr(2),
				TaskStartToCloseTimeoutSeconds:      common.Int32Ptr(1),
			},
		},
	)
	s.Nil(err)

	di := addDecisionTaskScheduledEvent(mutableState)

	timerTask := &persistence.TimerTaskInfo{
		Version:             s.version,
		DomainID:            s.domainID,
		WorkflowID:          workflowExecution.GetWorkflowId(),
		RunID:               workflowExecution.GetRunId(),
		TaskID:              int64(100),
		TaskType:            persistence.TaskTypeWorkflowBackoffTimer,
		VisibilityTimestamp: s.now,
	}

	persistenceMutableState := s.createPersistenceMutableState(mutableState, di.ScheduleID, di.Version)
	s.mockExecutionMgr.On("GetWorkflowExecution", mock.Anything).Return(&persistence.GetWorkflowExecutionResponse{State: persistenceMutableState}, nil).Once()

	s.mockShard.SetCurrentTime(s.clusterName, s.now)
	err = s.timerQueueStandbyTaskExecutor.execute(timerTask, true)
	s.Nil(err)
}

func (s *timerQueueStandbyTaskExecutorSuite) TestProcessWorkflowTimeout_Pending() {

	workflowExecution := workflow.WorkflowExecution{
		WorkflowId: common.StringPtr("some random workflow ID"),
		RunId:      common.StringPtr(uuid.New()),
	}
	workflowType := "some random workflow type"
	taskListName := "some random task list"

	mutableState := execution.NewMutableStateBuilderWithReplicationStateWithEventV2(s.mockShard, s.mockShard.GetEventsCache(), s.logger, s.version, workflowExecution.GetRunId(), testGlobalDomainEntry)
	_, err := mutableState.AddWorkflowExecutionStartedEvent(
		workflowExecution,
		&history.StartWorkflowExecutionRequest{
			DomainUUID: common.StringPtr(s.domainID),
			StartRequest: &workflow.StartWorkflowExecutionRequest{
				WorkflowType:                        &workflow.WorkflowType{Name: common.StringPtr(workflowType)},
				TaskList:                            &workflow.TaskList{Name: common.StringPtr(taskListName)},
				ExecutionStartToCloseTimeoutSeconds: common.Int32Ptr(2),
				TaskStartToCloseTimeoutSeconds:      common.Int32Ptr(1),
			},
		},
	)
	s.Nil(err)

	di := addDecisionTaskScheduledEvent(mutableState)
	startEvent := addDecisionTaskStartedEvent(mutableState, di.ScheduleID, taskListName, uuid.New())
	di.StartedID = startEvent.GetEventId()
	completionEvent := addDecisionTaskCompletedEvent(mutableState, di.ScheduleID, di.StartedID, nil, "some random identity")
	nextEventID := completionEvent.GetEventId() + 1

	timerTask := &persistence.TimerTaskInfo{
		Version:             s.version,
		DomainID:            s.domainID,
		WorkflowID:          workflowExecution.GetWorkflowId(),
		RunID:               workflowExecution.GetRunId(),
		TaskID:              int64(100),
		TaskType:            persistence.TaskTypeWorkflowTimeout,
		TimeoutType:         int(workflow.TimeoutTypeStartToClose),
		VisibilityTimestamp: s.now,
	}

	persistenceMutableState := s.createPersistenceMutableState(mutableState, completionEvent.GetEventId(), completionEvent.GetVersion())
	s.mockExecutionMgr.On("GetWorkflowExecution", mock.Anything).Return(&persistence.GetWorkflowExecutionResponse{State: persistenceMutableState}, nil).Once()

	s.mockShard.SetCurrentTime(s.clusterName, s.now)
	err = s.timerQueueStandbyTaskExecutor.execute(timerTask, true)
	s.Equal(ErrTaskRetry, err)

	s.mockShard.SetCurrentTime(s.clusterName, s.now.Add(s.fetchHistoryDuration))
	s.mockHistoryRereplicator.On("SendMultiWorkflowHistory",
		timerTask.DomainID, timerTask.WorkflowID,
		timerTask.RunID, nextEventID,
		timerTask.RunID, common.EndEventID,
	).Return(nil).Once()
	err = s.timerQueueStandbyTaskExecutor.execute(timerTask, true)
	s.Equal(ErrTaskRetry, err)

	s.mockShard.SetCurrentTime(s.clusterName, s.now.Add(s.discardDuration))
	err = s.timerQueueStandbyTaskExecutor.execute(timerTask, true)
	s.Equal(ErrTaskDiscarded, err)
}

func (s *timerQueueStandbyTaskExecutorSuite) TestProcessWorkflowTimeout_Success() {

	workflowExecution := workflow.WorkflowExecution{
		WorkflowId: common.StringPtr("some random workflow ID"),
		RunId:      common.StringPtr(uuid.New()),
	}
	workflowType := "some random workflow type"
	taskListName := "some random task list"

	mutableState := execution.NewMutableStateBuilderWithReplicationStateWithEventV2(s.mockShard, s.mockShard.GetEventsCache(), s.logger, s.version, workflowExecution.GetRunId(), testGlobalDomainEntry)
	_, err := mutableState.AddWorkflowExecutionStartedEvent(
		workflowExecution,
		&history.StartWorkflowExecutionRequest{
			DomainUUID: common.StringPtr(s.domainID),
			StartRequest: &workflow.StartWorkflowExecutionRequest{
				WorkflowType:                        &workflow.WorkflowType{Name: common.StringPtr(workflowType)},
				TaskList:                            &workflow.TaskList{Name: common.StringPtr(taskListName)},
				ExecutionStartToCloseTimeoutSeconds: common.Int32Ptr(2),
				TaskStartToCloseTimeoutSeconds:      common.Int32Ptr(1),
			},
		},
	)
	s.Nil(err)

	di := addDecisionTaskScheduledEvent(mutableState)
	event := addDecisionTaskStartedEvent(mutableState, di.ScheduleID, taskListName, uuid.New())
	di.StartedID = event.GetEventId()
	event = addDecisionTaskCompletedEvent(mutableState, di.ScheduleID, di.StartedID, nil, "some random identity")
	event = addCompleteWorkflowEvent(mutableState, event.GetEventId(), nil)

	timerTask := &persistence.TimerTaskInfo{
		Version:             s.version,
		DomainID:            s.domainID,
		WorkflowID:          workflowExecution.GetWorkflowId(),
		RunID:               workflowExecution.GetRunId(),
		TaskID:              int64(100),
		TaskType:            persistence.TaskTypeWorkflowTimeout,
		TimeoutType:         int(workflow.TimeoutTypeStartToClose),
		VisibilityTimestamp: s.now,
	}

	persistenceMutableState := s.createPersistenceMutableState(mutableState, event.GetEventId(), event.GetVersion())
	s.mockExecutionMgr.On("GetWorkflowExecution", mock.Anything).Return(&persistence.GetWorkflowExecutionResponse{State: persistenceMutableState}, nil).Once()

	s.mockShard.SetCurrentTime(s.clusterName, s.now)
	err = s.timerQueueStandbyTaskExecutor.execute(timerTask, true)
	s.Nil(err)
}

func (s *timerQueueStandbyTaskExecutorSuite) TestProcessRetryTimeout() {

	workflowExecution := workflow.WorkflowExecution{
		WorkflowId: common.StringPtr("some random workflow ID"),
		RunId:      common.StringPtr(uuid.New()),
	}
	workflowType := "some random workflow type"
	taskListName := "some random task list"

	mutableState := execution.NewMutableStateBuilderWithReplicationStateWithEventV2(s.mockShard, s.mockShard.GetEventsCache(), s.logger, s.version, workflowExecution.GetRunId(), testGlobalDomainEntry)
	_, err := mutableState.AddWorkflowExecutionStartedEvent(
		workflowExecution,
		&history.StartWorkflowExecutionRequest{
			DomainUUID: common.StringPtr(s.domainID),
			StartRequest: &workflow.StartWorkflowExecutionRequest{
				WorkflowType:                        &workflow.WorkflowType{Name: common.StringPtr(workflowType)},
				TaskList:                            &workflow.TaskList{Name: common.StringPtr(taskListName)},
				ExecutionStartToCloseTimeoutSeconds: common.Int32Ptr(2),
				TaskStartToCloseTimeoutSeconds:      common.Int32Ptr(1),
			},
		},
	)
	s.Nil(err)

	timerTask := &persistence.TimerTaskInfo{
		Version:             s.version,
		DomainID:            s.domainID,
		WorkflowID:          workflowExecution.GetWorkflowId(),
		RunID:               workflowExecution.GetRunId(),
		TaskID:              int64(100),
		TaskType:            persistence.TaskTypeActivityRetryTimer,
		TimeoutType:         int(workflow.TimeoutTypeStartToClose),
		VisibilityTimestamp: s.now,
	}

	s.mockShard.SetCurrentTime(s.clusterName, s.now)
	err = s.timerQueueStandbyTaskExecutor.execute(timerTask, true)
	s.Nil(err)
}

func (s *timerQueueStandbyTaskExecutorSuite) createPersistenceMutableState(
	ms execution.MutableState,
	lastEventID int64,
	lastEventVersion int64,
) *persistence.WorkflowMutableState {

	if ms.GetReplicationState() != nil {
		ms.UpdateReplicationStateLastEventID(lastEventVersion, lastEventID)
	} else if ms.GetVersionHistories() != nil {
		currentVersionHistory, err := ms.GetVersionHistories().GetCurrentVersionHistory()
		s.NoError(err)
		err = currentVersionHistory.AddOrUpdateItem(persistence.NewVersionHistoryItem(
			lastEventID, lastEventVersion,
		))
		s.NoError(err)
	}

	return execution.CreatePersistenceMutableState(ms)
}
