package models

type CommandState struct {
	LoggedIn      bool
	CurrRepoKnown bool
	User          *User
	Organization  *Organization
	Project       *Project
}

// SetupRequirement defines what level of setup is needed for each component.
type SetupRequirement int

const (
	Ignore             SetupRequirement = iota
	NeedRepoLookup                      // we need to lookup the project from the git repo
	NeedIDOnly                          // we only need the id
	NeedExistingIDOnly                  // need id but can't create new one
	NeedData                            // we need all the data, can create new one
	NeedExistingData                    // we must have all the data but we can't create a new one
	MustNotExist                        // we must not have this
)

// LoginRequirement defines what level of login is needed.
type LoginRequirement int

const (
	IgnoreLogin LoginRequirement = iota // don't care if we are logged in or not
	NeedLogin
)

// SetupRequest defines what a command needs to be properly initialized.
type SetupRequest struct {
	LoginRequired        LoginRequirement
	OrganizationRequired SetupRequirement
	ProjectRequired      SetupRequirement
}
