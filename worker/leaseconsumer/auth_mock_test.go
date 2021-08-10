// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/juju/juju/apiserver/httpcontext (interfaces: Authenticator)

// Package leaseconsumer is a generated GoMock package.
package leaseconsumer

import (
	context "context"
	http "net/http"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	httpcontext "github.com/juju/juju/apiserver/httpcontext"
	params "github.com/juju/juju/apiserver/params"
)

// MockAuthenticator is a mock of Authenticator interface.
type MockAuthenticator struct {
	ctrl     *gomock.Controller
	recorder *MockAuthenticatorMockRecorder
}

// MockAuthenticatorMockRecorder is the mock recorder for MockAuthenticator.
type MockAuthenticatorMockRecorder struct {
	mock *MockAuthenticator
}

// NewMockAuthenticator creates a new mock instance.
func NewMockAuthenticator(ctrl *gomock.Controller) *MockAuthenticator {
	mock := &MockAuthenticator{ctrl: ctrl}
	mock.recorder = &MockAuthenticatorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAuthenticator) EXPECT() *MockAuthenticatorMockRecorder {
	return m.recorder
}

// Authenticate mocks base method.
func (m *MockAuthenticator) Authenticate(arg0 *http.Request) (httpcontext.AuthInfo, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Authenticate", arg0)
	ret0, _ := ret[0].(httpcontext.AuthInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Authenticate indicates an expected call of Authenticate.
func (mr *MockAuthenticatorMockRecorder) Authenticate(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Authenticate", reflect.TypeOf((*MockAuthenticator)(nil).Authenticate), arg0)
}

// AuthenticateLoginRequest mocks base method.
func (m *MockAuthenticator) AuthenticateLoginRequest(arg0 context.Context, arg1, arg2 string, arg3 params.LoginRequest) (httpcontext.AuthInfo, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AuthenticateLoginRequest", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(httpcontext.AuthInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// AuthenticateLoginRequest indicates an expected call of AuthenticateLoginRequest.
func (mr *MockAuthenticatorMockRecorder) AuthenticateLoginRequest(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AuthenticateLoginRequest", reflect.TypeOf((*MockAuthenticator)(nil).AuthenticateLoginRequest), arg0, arg1, arg2, arg3)
}
