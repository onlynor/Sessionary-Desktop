package shellenv

// ResolvePath: a Windows process gets the user's PATH from the registry, the same one a terminal gets.
func ResolvePath(current string) string { return current }
