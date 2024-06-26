// Code generated by mockery v2.43.2. DO NOT EDIT.

package blackbox

import (
	context "context"
	blackbox "shortener/proto/blackbox"

	grpc "google.golang.org/grpc"

	mock "github.com/stretchr/testify/mock"
)

// MockBlackboxServiceClient is an autogenerated mock type for the BlackboxServiceClient type
type MockBlackboxServiceClient struct {
	mock.Mock
}

type MockBlackboxServiceClient_Expecter struct {
	mock *mock.Mock
}

func (_m *MockBlackboxServiceClient) EXPECT() *MockBlackboxServiceClient_Expecter {
	return &MockBlackboxServiceClient_Expecter{mock: &_m.Mock}
}

// IssueToken provides a mock function with given fields: ctx, in, opts
func (_m *MockBlackboxServiceClient) IssueToken(ctx context.Context, in *blackbox.IssueTokenReq, opts ...grpc.CallOption) (*blackbox.IssueTokenRsp, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for IssueToken")
	}

	var r0 *blackbox.IssueTokenRsp
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *blackbox.IssueTokenReq, ...grpc.CallOption) (*blackbox.IssueTokenRsp, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *blackbox.IssueTokenReq, ...grpc.CallOption) *blackbox.IssueTokenRsp); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*blackbox.IssueTokenRsp)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *blackbox.IssueTokenReq, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockBlackboxServiceClient_IssueToken_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'IssueToken'
type MockBlackboxServiceClient_IssueToken_Call struct {
	*mock.Call
}

// IssueToken is a helper method to define mock.On call
//   - ctx context.Context
//   - in *blackbox.IssueTokenReq
//   - opts ...grpc.CallOption
func (_e *MockBlackboxServiceClient_Expecter) IssueToken(ctx interface{}, in interface{}, opts ...interface{}) *MockBlackboxServiceClient_IssueToken_Call {
	return &MockBlackboxServiceClient_IssueToken_Call{Call: _e.mock.On("IssueToken",
		append([]interface{}{ctx, in}, opts...)...)}
}

func (_c *MockBlackboxServiceClient_IssueToken_Call) Run(run func(ctx context.Context, in *blackbox.IssueTokenReq, opts ...grpc.CallOption)) *MockBlackboxServiceClient_IssueToken_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]grpc.CallOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(grpc.CallOption)
			}
		}
		run(args[0].(context.Context), args[1].(*blackbox.IssueTokenReq), variadicArgs...)
	})
	return _c
}

func (_c *MockBlackboxServiceClient_IssueToken_Call) Return(_a0 *blackbox.IssueTokenRsp, _a1 error) *MockBlackboxServiceClient_IssueToken_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockBlackboxServiceClient_IssueToken_Call) RunAndReturn(run func(context.Context, *blackbox.IssueTokenReq, ...grpc.CallOption) (*blackbox.IssueTokenRsp, error)) *MockBlackboxServiceClient_IssueToken_Call {
	_c.Call.Return(run)
	return _c
}

// ValidateToken provides a mock function with given fields: ctx, in, opts
func (_m *MockBlackboxServiceClient) ValidateToken(ctx context.Context, in *blackbox.ValidateTokenReq, opts ...grpc.CallOption) (*blackbox.ValidateTokenRsp, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for ValidateToken")
	}

	var r0 *blackbox.ValidateTokenRsp
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *blackbox.ValidateTokenReq, ...grpc.CallOption) (*blackbox.ValidateTokenRsp, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *blackbox.ValidateTokenReq, ...grpc.CallOption) *blackbox.ValidateTokenRsp); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*blackbox.ValidateTokenRsp)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *blackbox.ValidateTokenReq, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockBlackboxServiceClient_ValidateToken_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ValidateToken'
type MockBlackboxServiceClient_ValidateToken_Call struct {
	*mock.Call
}

// ValidateToken is a helper method to define mock.On call
//   - ctx context.Context
//   - in *blackbox.ValidateTokenReq
//   - opts ...grpc.CallOption
func (_e *MockBlackboxServiceClient_Expecter) ValidateToken(ctx interface{}, in interface{}, opts ...interface{}) *MockBlackboxServiceClient_ValidateToken_Call {
	return &MockBlackboxServiceClient_ValidateToken_Call{Call: _e.mock.On("ValidateToken",
		append([]interface{}{ctx, in}, opts...)...)}
}

func (_c *MockBlackboxServiceClient_ValidateToken_Call) Run(run func(ctx context.Context, in *blackbox.ValidateTokenReq, opts ...grpc.CallOption)) *MockBlackboxServiceClient_ValidateToken_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]grpc.CallOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(grpc.CallOption)
			}
		}
		run(args[0].(context.Context), args[1].(*blackbox.ValidateTokenReq), variadicArgs...)
	})
	return _c
}

func (_c *MockBlackboxServiceClient_ValidateToken_Call) Return(_a0 *blackbox.ValidateTokenRsp, _a1 error) *MockBlackboxServiceClient_ValidateToken_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockBlackboxServiceClient_ValidateToken_Call) RunAndReturn(run func(context.Context, *blackbox.ValidateTokenReq, ...grpc.CallOption) (*blackbox.ValidateTokenRsp, error)) *MockBlackboxServiceClient_ValidateToken_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockBlackboxServiceClient creates a new instance of MockBlackboxServiceClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockBlackboxServiceClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockBlackboxServiceClient {
	mock := &MockBlackboxServiceClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
