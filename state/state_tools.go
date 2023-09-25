package state

//when you create a new project the cli must know where it lives in order to manipulate it
//there must be some .config file in the directory you're in so that the cli knows what to do
//in order to manipulate it.

func SaveState(key string, value string, path string) error {
	return nil
}

func GetState(key string, path string) string {
	return ""
}
