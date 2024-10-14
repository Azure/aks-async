// Code generated by MockGen. DO NOT EDIT.
// Source: servicebus_interface.go
//
// Generated by this command:
//
//	mockgen -source=servicebus_interface.go -destination=../mocks/mock_service_bus.go -package=mocks
//

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	servicebus "github.com/Azure/aks-async/servicebus"
	gomock "go.uber.org/mock/gomock"
)

// MockServiceBusClientInterface is a mock of ServiceBusClientInterface interface.
type MockServiceBusClientInterface struct {
	ctrl     *gomock.Controller
	recorder *MockServiceBusClientInterfaceMockRecorder
}

// MockServiceBusClientInterfaceMockRecorder is the mock recorder for MockServiceBusClientInterface.
type MockServiceBusClientInterfaceMockRecorder struct {
	mock *MockServiceBusClientInterface
}

// NewMockServiceBusClientInterface creates a new mock instance.
func NewMockServiceBusClientInterface(ctrl *gomock.Controller) *MockServiceBusClientInterface {
	mock := &MockServiceBusClientInterface{ctrl: ctrl}
	mock.recorder = &MockServiceBusClientInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockServiceBusClientInterface) EXPECT() *MockServiceBusClientInterfaceMockRecorder {
	return m.recorder
}

// NewServiceBusReceiver mocks base method.
func (m *MockServiceBusClientInterface) NewServiceBusReceiver(ctx context.Context, topicOrQueue string) (servicebus.ReceiverInterface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewServiceBusReceiver", ctx, topicOrQueue)
	ret0, _ := ret[0].(servicebus.ReceiverInterface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewServiceBusReceiver indicates an expected call of NewServiceBusReceiver.
func (mr *MockServiceBusClientInterfaceMockRecorder) NewServiceBusReceiver(ctx, topicOrQueue any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewServiceBusReceiver", reflect.TypeOf((*MockServiceBusClientInterface)(nil).NewServiceBusReceiver), ctx, topicOrQueue)
}

// NewServiceBusSender mocks base method.
func (m *MockServiceBusClientInterface) NewServiceBusSender(ctx context.Context, queue string) (servicebus.SenderInterface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewServiceBusSender", ctx, queue)
	ret0, _ := ret[0].(servicebus.SenderInterface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewServiceBusSender indicates an expected call of NewServiceBusSender.
func (mr *MockServiceBusClientInterfaceMockRecorder) NewServiceBusSender(ctx, queue any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewServiceBusSender", reflect.TypeOf((*MockServiceBusClientInterface)(nil).NewServiceBusSender), ctx, queue)
}

// MockSenderInterface is a mock of SenderInterface interface.
type MockSenderInterface struct {
	ctrl     *gomock.Controller
	recorder *MockSenderInterfaceMockRecorder
}

// MockSenderInterfaceMockRecorder is the mock recorder for MockSenderInterface.
type MockSenderInterfaceMockRecorder struct {
	mock *MockSenderInterface
}

// NewMockSenderInterface creates a new mock instance.
func NewMockSenderInterface(ctrl *gomock.Controller) *MockSenderInterface {
	mock := &MockSenderInterface{ctrl: ctrl}
	mock.recorder = &MockSenderInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSenderInterface) EXPECT() *MockSenderInterfaceMockRecorder {
	return m.recorder
}

// SendMessage mocks base method.
func (m *MockSenderInterface) SendMessage(ctx context.Context, message []byte) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendMessage", ctx, message)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendMessage indicates an expected call of SendMessage.
func (mr *MockSenderInterfaceMockRecorder) SendMessage(ctx, message any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendMessage", reflect.TypeOf((*MockSenderInterface)(nil).SendMessage), ctx, message)
}

// MockReceiverInterface is a mock of ReceiverInterface interface.
type MockReceiverInterface struct {
	ctrl     *gomock.Controller
	recorder *MockReceiverInterfaceMockRecorder
}

// MockReceiverInterfaceMockRecorder is the mock recorder for MockReceiverInterface.
type MockReceiverInterfaceMockRecorder struct {
	mock *MockReceiverInterface
}

// NewMockReceiverInterface creates a new mock instance.
func NewMockReceiverInterface(ctrl *gomock.Controller) *MockReceiverInterface {
	mock := &MockReceiverInterface{ctrl: ctrl}
	mock.recorder = &MockReceiverInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockReceiverInterface) EXPECT() *MockReceiverInterfaceMockRecorder {
	return m.recorder
}

// ReceiveMessage mocks base method.
func (m *MockReceiverInterface) ReceiveMessage(ctx context.Context) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReceiveMessage", ctx)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReceiveMessage indicates an expected call of ReceiveMessage.
func (mr *MockReceiverInterfaceMockRecorder) ReceiveMessage(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReceiveMessage", reflect.TypeOf((*MockReceiverInterface)(nil).ReceiveMessage), ctx)
}
