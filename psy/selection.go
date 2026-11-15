package psy

// Selection is the type of score selection.
type Selection string

const (
	First Selection = "first"
	Last  Selection = "last"
	All   Selection = "all"
	None  Selection = "none"
)

// String returns the string representation of the Selection.
func (s Selection) String() string {
	return string(s)
}

// IsValid returns true if the Selection is valid.
func (s Selection) IsValid() bool {
	return s == First || s == Last || s == All || s == None
}
