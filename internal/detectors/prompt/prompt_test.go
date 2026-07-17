package prompt

import (
	"testing"

	"github.com/airomhq/airom/pkg/airom/detectortest"
)

func TestPrompt(t *testing.T) {
	detectortest.Run(t, NewPrompt(), detectortest.Fixtures{Dir: "testdata"})
}
