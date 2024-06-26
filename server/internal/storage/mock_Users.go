// Code generated by mockery v2.43.2. DO NOT EDIT.

package storage

import (
	context "context"
	responses "shortener/pkg/responses"

	mock "github.com/stretchr/testify/mock"
)

// MockUsers is an autogenerated mock type for the Users type
type MockUsers struct {
	mock.Mock
}

type MockUsers_Expecter struct {
	mock *mock.Mock
}

func (_m *MockUsers) EXPECT() *MockUsers_Expecter {
	return &MockUsers_Expecter{mock: &_m.Mock}
}

// Insert provides a mock function with given fields: ctx, rr
func (_m *MockUsers) Insert(ctx context.Context, rr []*responses.Authenticator) {
	_m.Called(ctx, rr)
}

// MockUsers_Insert_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Insert'
type MockUsers_Insert_Call struct {
	*mock.Call
}

// Insert is a helper method to define mock.On call
//   - ctx context.Context
//   - rr []*responses.Authenticator
func (_e *MockUsers_Expecter) Insert(ctx interface{}, rr interface{}) *MockUsers_Insert_Call {
	return &MockUsers_Insert_Call{Call: _e.mock.On("Insert", ctx, rr)}
}

func (_c *MockUsers_Insert_Call) Run(run func(ctx context.Context, rr []*responses.Authenticator)) *MockUsers_Insert_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].([]*responses.Authenticator))
	})
	return _c
}

func (_c *MockUsers_Insert_Call) Return() *MockUsers_Insert_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockUsers_Insert_Call) RunAndReturn(run func(context.Context, []*responses.Authenticator)) *MockUsers_Insert_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockUsers creates a new instance of MockUsers. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockUsers(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockUsers {
	mock := &MockUsers{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
