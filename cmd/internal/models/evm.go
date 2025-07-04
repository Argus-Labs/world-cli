package models

type StartEVMFlags struct {
	Config      string
	DAAuthToken string
	UseDevDA    bool
}

type StopEVMFlags struct {
	Config string
}
