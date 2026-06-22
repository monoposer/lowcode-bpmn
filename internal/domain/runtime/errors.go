package runtime

import "errors"

var ErrVersionConflict = errors.New("process instance was modified concurrently")
