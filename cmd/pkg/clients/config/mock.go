package config

import "github.com/stretchr/testify/mock"

var _ ClientInterface = (*MockClient)(nil)

type MockClient struct {
	mock.Mock
}

func (m *MockClient) GetConfig() *Config {
	args := m.Called()
	return args.Get(0).(*Config)
}

func (m *MockClient) Save() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockClient) AddKnownProject(pID, oID, rURL, rPath, pName string) {
	m.Called(pID, oID, rURL, rPath, pName)
}
