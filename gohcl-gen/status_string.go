package GoHclGen

import "fmt"

func (c Color) String() string {
	switch c {
	case Write:
		return "Write"
	case Red:
		return "Red"
	case Blue:
		return "Blue"
	}
	return fmt.Sprintf("Status(%d)", c)
}

func (c Status) String() string {
	switch c {
	case Offline:
		return "Offline"
	case Online:
		return "Online"
	case Disable:
		return "Disable"
	case Added:
		return "Added"
	case Deleted:
		return "Deleted"
	}
	return fmt.Sprintf("Status(%d)", c)
}
