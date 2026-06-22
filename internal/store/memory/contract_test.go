package memory

import (
	"testing"

	porttesting "github.com/monoposer/lowcode-bpmn/internal/domain/ports/testing"
)

func TestProcessRepositoryContract(t *testing.T) {
	porttesting.RunProcessRepositoryContract(t, NewStore())
}
