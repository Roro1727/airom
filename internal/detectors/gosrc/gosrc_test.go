package gosrc

import (
	"testing"

	"github.com/airomhq/airom/pkg/airom/detectortest"
)

func TestGoSource(t *testing.T) {
	detectortest.Run(t, NewGoSource(), detectortest.Fixtures{Dir: "testdata"})
}
