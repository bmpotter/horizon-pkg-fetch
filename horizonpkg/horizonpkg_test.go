// +build integration

package horizonpkg

import (
	"testing"
)

func Test_DockerImagePkgBuilder_Suite(t *testing.T) {
	author := "someguy@overthar.it"

	builder, err := NewDockerImagePkgBuilder(FILE, author, []string{})
	if err != nil {
		t.Error("Failed to build DockerImagePkgBuilder instance")
	}

	t.Run("DockerImagePkgBuilder produces empty pkg w/ proper meta", func(t *testing.T) {

		p, _, _ := builder.Build()
		if p.Meta.PartsType != FILE ||
			p.Meta.Author != author ||
			p.Meta.SpecVersion != specVersion ||
			p.Meta.Provides.ProvidesType != DOCKER ||
			len(p.Meta.Provides.Images) != 0 ||
			len(p.Parts) != 0 {
			t.Errorf("Improperly configured Pkg returned by builder: %v", p)
		}
	})

	t.Run("DockerImagePkgBuilder.AddPart() checks sha1sum for length", func(t *testing.T) {
		_, err := builder.AddPart("", "1222", "someimage:latest", []string{"foo"}, 33, PartSource{"https://goo.foo"})

		if err == nil {
			t.Errorf("Builder failed to check sha1sum for length")
		}
	})

	t.Run("DockerImagePkgBuilder.AddPart() checks sha1sum for content", func(t *testing.T) {
		_, err := builder.AddPart("", "123456789012345678901234567890123456789#", "someimage:latest", []string{"foo"}, 33, PartSource{"https://goo.foo"})

		if err == nil {
			t.Errorf("Builder failed to check sha1sum for content")
		}
	})

	t.Run("DockerImagePkgBuilder.AddPart() disallows empty signatures if builder is configured with defaults", func(t *testing.T) {
		_, err := builder.AddPart("", "1234567890123456789012345678901234567890", "someimage:latest", []string{}, 33, PartSource{"https://goo.foo"})

		if err == nil {
			t.Errorf("Builder allowed empty signatures when adding part and shouldn't have")
		}
	})

	t.Run("DockerImagePkgBuilder.AddPart() permits empty signatures for part when builder is so configured", func(t *testing.T) {
		unsecureBuilder, _ := NewDockerImagePkgBuilder(FILE, author, []string{"someimage:latest"})
		unsecureBuilder.SetPermitEmptySignatures()
		_, err := unsecureBuilder.AddPart("", "1234567890123456789012345678901234567890123456789012345678901234", "someimage:latest", []string{}, 33, PartSource{"https://goo.foo"})

		if err != nil {
			t.Logf("%v", err)
			t.Errorf("Builder did not permit empty signatures when adding part but it should have")
		}
	})
}
