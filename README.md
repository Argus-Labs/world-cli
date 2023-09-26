# world-engine-cli
world-engine-cli

temp Dev notes:

install this:
`go install github.com/spf13/cobra-cli@latest`

Then run:
`cobra-cli add create -p 'configCmd` to add a command line parameter "create" with an option "-p" that takes an argument 'configCmd'

see newProject.go to see how this utilizes these libraries to construct an interface:
- https://github.com/charmbracelet/lipgloss for terminal rendering
- https://github.com/charmbracelet/bubbletea for the tui framework
- https://github.com/charmbracelet/bubbles for tui widgets that work with bubbletea

All this does is clone the starter-game-template. 

Test with these commands; in the project dir run:
1. `go build`
2. `./world-engine-cli new-project myproject`
   


