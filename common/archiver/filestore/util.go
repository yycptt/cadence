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

package filestore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/dgryski/go-farm"

	"github.com/uber/cadence/.gen/go/shared"
	"github.com/uber/cadence/common/archiver"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/log/tag"
)

const (
	dirMode  = os.FileMode(0700)
	fileMode = os.FileMode(0600)
)

var (
	errDirectoryExpected = errors.New("a path to a directory was expected")
	errFileExpected      = errors.New("a path to a file was expected")
)

func fileExists(filepath string) (bool, error) {
	if info, err := os.Stat(filepath); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	} else if info.IsDir() {
		return false, errFileExpected
	}
	return true, nil
}

func directoryExists(path string) (bool, error) {
	if info, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	} else if !info.IsDir() {
		return false, errDirectoryExpected
	}
	return true, nil
}

func mkdirAll(path string) error {
	return os.MkdirAll(path, dirMode)
}

func writeFile(filepath string, data []byte) error {
	if err := os.Remove(filepath); err != nil && !os.IsNotExist(err) {
		return err
	}
	f, err := os.Create(filepath)
	defer f.Close()
	if err != nil {
		return err
	}
	if err = f.Chmod(fileMode); err != nil {
		return err
	}
	if _, err = f.Write(data); err != nil {
		return err
	}
	return nil
}

func readFile(filepath string) ([]byte, error) {
	return ioutil.ReadFile(filepath)
}

func deleteFile(filepath string) (bool, error) {
	if err := os.Remove(filepath); err != nil {
		return false, err
	}
	return true, nil
}

func listFiles(path string) ([]string, error) {
	if info, err := os.Stat(path); err != nil {
		return nil, err
	} else if !info.IsDir() {
		return nil, errDirectoryExpected
	}

	children, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, c := range children {
		if c.IsDir() {
			continue
		}
		files = append(files, c.Name())
	}
	return files, nil
}

func encodeHistoryBatches(historyBatches []*shared.History) ([]byte, error) {
	data, err := json.Marshal(historyBatches)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func decodeHistoryBatches(data []byte) ([]*shared.History, error) {
	historyBatches := []*shared.History{}
	err := json.Unmarshal(data, &historyBatches)
	if err != nil {
		return nil, err
	}
	return historyBatches, nil
}

func tagLoggerWithArchiveHistoryRequest(logger log.Logger, request *archiver.ArchiveHistoryRequest) log.Logger {
	return logger.WithTags(
		tag.ShardID(request.ShardID),
		tag.ArchivalRequestDomainID(request.DomainID),
		tag.ArchivalRequestDomainName(request.DomainName),
		tag.ArchivalRequestWorkflowID(request.WorkflowID),
		tag.ArchivalRequestRunID(request.RunID),
		tag.ArchivalRequestEventStoreVersion(request.EventStoreVersion),
		tag.ArchivalRequestBranchToken(request.BranchToken),
		tag.ArchivalRequestNextEventID(request.NextEventID),
		tag.ArchivalRequestCloseFailoverVersion(request.CloseFailoverVersion),
	)
}

func contextExpired(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

func getDirPathFromURI(URI string) string {
	return URI[len(fileStoreScheme):]
}

func constructFilename(request *archiver.ArchiveHistoryRequest) string {
	combinedHash := constructFilenamePrefix(request)
	return fmt.Sprintf("%s-%v.history", combinedHash, request.CloseFailoverVersion)
}

func constructFilenamePrefix(request *archiver.ArchiveHistoryRequest) string {
	domainIDHash := fmt.Sprintf("%v", farm.Fingerprint64([]byte(request.DomainID)))
	workflowIDHash := fmt.Sprintf("%v", farm.Fingerprint64([]byte(request.WorkflowID)))
	runIDHash := fmt.Sprintf("%v", farm.Fingerprint64([]byte(request.RunID)))
	return strings.Join([]string{domainIDHash, workflowIDHash, runIDHash}, "")
}

func historyMutated(request *archiver.ArchiveHistoryRequest, historyBatches []*shared.History, isLast bool) bool {
	lastBatch := historyBatches[len(historyBatches)-1].Events
	lastEvent := lastBatch[len(lastBatch)-1]
	lastFailoverVersion := lastEvent.GetVersion()
	if lastFailoverVersion > request.CloseFailoverVersion {
		return true
	}

	if !isLast {
		return false
	}
	lastEventID := lastEvent.GetEventId()
	return lastFailoverVersion != request.CloseFailoverVersion || lastEventID+1 != request.NextEventID
}