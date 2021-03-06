// Copyright 2015 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.
//
// Author: Vivek Menezes (vivek@cockroachlabs.com)

package sql

import (
	"time"

	"golang.org/x/net/trace"

	"github.com/cockroachdb/cockroach/roachpb"
	"github.com/cockroachdb/cockroach/util"
)

type isSessionTimezone interface {
	isSessionTimezone()
}

// SessionLocation ...
type SessionLocation struct {
	Location string
}

// SessionOffset ...
type SessionOffset struct {
	Offset int64
}

func (*SessionLocation) isSessionTimezone() {}
func (*SessionOffset) isSessionTimezone()   {}

// TxnStateEnum represents the state of a SQL txn.
type TxnStateEnum int

//go:generate stringer -type=TxnStateEnum
const (
	// No txn is in scope. Either there never was one, or it got committed/rolled back.
	NoTxn TxnStateEnum = iota
	// A txn is in scope.
	Open
	// The txn has encoutered a (non-retriable) error.
	// Statements will be rejected until a COMMIT/ROLLBACK is seen.
	Aborted
	// The txn has encoutered a retriable error.
	// Statements will be rejected until a RESTART_TRANSACTION is seen.
	RestartWait
	// The KV txn has been committed successfully through a RELEASE.
	// Statements are rejected until a COMMIT is seen.
	CommitWait
)

// SessionTransaction ...
type SessionTransaction struct {
	// If nil, it means we're not inside a (KV) txn.
	Txn          *roachpb.Transaction
	State        TxnStateEnum
	retryIntent  bool
	UserPriority roachpb.UserPriority
	// Indicates that the transaction is mutating keys in the SystemConfig span.
	MutatesSystemConfig bool
}

// Session ...
type Session struct {
	Database string
	Syntax   int32
	// Info about the open transaction (if any).
	// TODO(andrei): get rid of SessionTransaction; store the txnState directly.
	Txn                   SessionTransaction
	Timezone              isSessionTimezone
	DefaultIsolationLevel roachpb.IsolationType
	Trace                 trace.Trace
}

func (s *Session) getLocation() (*time.Location, error) {
	switch t := s.Timezone.(type) {
	case nil:
		return time.UTC, nil
	case *SessionLocation:
		// TODO(vivek): Cache the location.
		return time.LoadLocation(t.Location)
	case *SessionOffset:
		return time.FixedZone("", int(t.Offset)), nil
	default:
		return nil, util.Errorf("unhandled timezone variant type %T", t)
	}
}
