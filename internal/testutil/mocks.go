// SPDX-License-Identifier: Apache-2.0

package testutil

import (
	"github.com/kusari-oss/darn/internal/core/action"
	"github.com/stretchr/testify/mock"
)

// MockAction provides a versatile mock implementation of the Action interface
// This can be used for both file action tests and factory tests
type MockAction struct {
	mock.Mock
	Config  action.Config
	Context action.ActionContext
}

// Execute mocks the Execute method
func (m *MockAction) Execute(params map[string]interface{}) error {
	// If expectations are set, use those
	if m.Mock.ExpectedCalls != nil && len(m.Mock.ExpectedCalls) > 0 {
		args := m.Called(params)
		return args.Error(0)
	}

	// Otherwise, behave like a simple no-op action
	return nil
}

// Description returns the action description
func (m *MockAction) Description() string {
	// If expectations are set, use those
	if m.Mock.ExpectedCalls != nil && len(m.Mock.ExpectedCalls) > 0 {
		args := m.Called()
		return args.String(0)
	}

	// Otherwise, return from the config
	return m.Config.Description
}

// MockOutputAction extends MockAction to also implement the OutputAction interface
type MockOutputAction struct {
	MockAction
}

// ExecuteWithOutput mocks the ExecuteWithOutput method
func (m *MockOutputAction) ExecuteWithOutput(params map[string]interface{}) (map[string]interface{}, error) {
	args := m.Called(params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

// NewMockActionCreator returns a factory function for creating MockActions
// This is useful for registering with the factory
func NewMockActionCreator() func(config action.Config, ctx action.ActionContext) (action.Action, error) {
	return func(config action.Config, ctx action.ActionContext) (action.Action, error) {
		return &MockAction{
			Config:  config,
			Context: ctx,
		}, nil
	}
}
