package config

import "github.com/stretchr/testify/mock"

var _ ServiceInterface = (*MockService)(nil)

type MockService struct {
	mock.Mock
}

func (m *MockService) GetConfig() *Config {
	args := m.Called()
	return args.Get(0).(*Config)
}

func (m *MockService) Save() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockService) AddKnownProject(pID, oID, rURL, rPath, pName string) {
	m.Called(pID, oID, rURL, rPath, pName)
}
