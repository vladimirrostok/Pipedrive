// Code generated by MockGen. DO NOT EDIT.
// Source: reader.go
//
// Generated by this command:
//
//      mockgen -source=reader.go
//

// Package mock_reader is a generated GoMock package.
package mock_reader

import (
        model "pipedrive-sync/pipedrive-sync/internal/model"
        reflect "reflect"

        gomock "go.uber.org/mock/gomock"
)

// MockDataReader is a mock of DataReader interface.
type MockDataReader struct {
        ctrl     *gomock.Controller
        recorder *MockDataReaderMockRecorder
}

// MockDataReaderMockRecorder is the mock recorder for MockDataReader.
type MockDataReaderMockRecorder struct {
        mock *MockDataReader
}

// NewMockDataReader creates a new mock instance.
func NewMockDataReader(ctrl *gomock.Controller) *MockDataReader {
        mock := &MockDataReader{ctrl: ctrl}
        mock.recorder = &MockDataReaderMockRecorder{mock}
        return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDataReader) EXPECT() *MockDataReaderMockRecorder {
        return m.recorder
}

// ReadCustomers mocks base method.
func (m *MockDataReader) ReadCustomers(filepath string, batchSize int) chan []model.Customer {
        m.ctrl.T.Helper()
        ret := m.ctrl.Call(m, "ReadCustomers", filepath, batchSize)
        ret0, _ := ret[0].(chan []model.Customer)
        return ret0
}

// ReadCustomers indicates an expected call of ReadCustomers.
func (mr *MockDataReaderMockRecorder) ReadCustomers(filepath, batchSize any) *gomock.Call {
        mr.mock.ctrl.T.Helper()
        return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadCustomers", reflect.TypeOf((*MockDataReader)(nil).ReadCustomers), filepath, batchSize)
}

// ReadDeals mocks base method.
func (m *MockDataReader) ReadDeals(filepath string, batchSize int) chan []model.Deal {
        m.ctrl.T.Helper()
        ret := m.ctrl.Call(m, "ReadDeals", filepath, batchSize)
        ret0, _ := ret[0].(chan []model.Deal)
        return ret0
}

// ReadDeals indicates an expected call of ReadDeals.
func (mr *MockDataReaderMockRecorder) ReadDeals(filepath, batchSize any) *gomock.Call {
        mr.mock.ctrl.T.Helper()
        return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadDeals", reflect.TypeOf((*MockDataReader)(nil).ReadDeals), filepath, batchSize)
}

// ReadOrders mocks base method.
func (m *MockDataReader) ReadOrders(filepath string, batchSize int) chan []model.Order {
        m.ctrl.T.Helper()
        ret := m.ctrl.Call(m, "ReadOrders", filepath, batchSize)
        ret0, _ := ret[0].(chan []model.Order)
        return ret0
}

// ReadOrders indicates an expected call of ReadOrders.
func (mr *MockDataReaderMockRecorder) ReadOrders(filepath, batchSize any) *gomock.Call {
        mr.mock.ctrl.T.Helper()
        return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadOrders", reflect.TypeOf((*MockDataReader)(nil).ReadOrders), filepath, batchSize)
}

// ReadPayments mocks base method.
func (m *MockDataReader) ReadPayments(filepath string, batchSize int) chan []model.Payment {
        m.ctrl.T.Helper()
        ret := m.ctrl.Call(m, "ReadPayments", filepath, batchSize)
        ret0, _ := ret[0].(chan []model.Payment)
        return ret0
}

// ReadPayments indicates an expected call of ReadPayments.
func (mr *MockDataReaderMockRecorder) ReadPayments(filepath, batchSize any) *gomock.Call {
        mr.mock.ctrl.T.Helper()
        return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadPayments", reflect.TypeOf((*MockDataReader)(nil).ReadPayments), filepath, batchSize)
}