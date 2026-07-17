package infra

import (
	"testing"

	"github.com/airomhq/airom/pkg/airom/detectortest"
)

func TestDockerfile(t *testing.T) {
	detectortest.Run(t, NewDockerfile(), detectortest.Fixtures{Dir: "testdata/dockerfile"})
}

func TestCompose(t *testing.T) {
	detectortest.Run(t, NewCompose(), detectortest.Fixtures{Dir: "testdata/compose"})
}
