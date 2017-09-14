package horizonpkg

import (
	"errors"
	"fmt"
	"github.com/golang/glog"
	"regexp"
	"strings"
)

type DockerImageParts map[string]DockerImagePart

type Pkg struct {
	Meta  *Meta            `json:"meta"`
	Parts DockerImageParts `json:"parts"`
}

type PkgBuilder struct {
	pkg                   *Pkg
	permitEmptySignatures bool
}

// NewDockerImagePkgBuilder is a factory method for a pkg builder. It's
// expected that after getting a reference to this type, one will use AddPart()
// and related functions to populate it and then call BuildPkg() to produce an
// immutable package.
func NewDockerImagePkgBuilder(partsType PartsType, author string, specVer string) (*PkgBuilder, error) {

	switch partsType {
	// TODO: could have a type like "REGISTRY" here that merely fetches from a docker registry
	case FILE:
		glog.V(5).Infof("Building DockerImagePkg with parts of type %v", partsType)
	default:
		return nil, fmt.Errorf("Unknown partsType: %v", partsType)
	}

	provides := DockerPartsProvides{
		ProvidesType: DOCKER,
		Images:       DockerImagePartNames{},
	}

	return &PkgBuilder{
		pkg: &Pkg{
			Meta: &Meta{
				PartsType:   partsType,
				Author:      author,
				SpecVersion: specVer,
				Provides:    provides,
			},
			Parts: DockerImageParts{},
		},
		permitEmptySignatures: false,
	}, nil
}

// SetPermitEmptySignatures sets this builder instance to allow empty
// signatures per-part.
func (p *PkgBuilder) SetPermitEmptySignatures() *PkgBuilder {
	// TODO: make sure there is a setting in Anax and the package builder tool to explicitly allow no signatures on parts otherwise an error is thrown
	p.permitEmptySignatures = true
	glog.Infof("Warning: permitEmptySignatures set on PkgBuilder instance with meta: %v", p.pkg.Meta)
	return p
}

// AddPart adds a DockerImagePart to Pkg.Parts and Pkg.Meta.Provides. Note
// that the id is not required; if it is an empty string, the sha1sum will be
// used as the id instead. A valid sha1sum is a hex representation of the
// 160-bit sequence.
func (p *PkgBuilder) AddPart(id string, sha1sum string, dockerImageRepoTag string, signatures []string, bytes uint, sources ...PartSource) (*PkgBuilder, error) {

	if sha1sumInvalid, err := regexp.MatchString("[^0-9A-Za-z]", sha1sum); sha1sumInvalid || err != nil || len(sha1sum) != 40 {
		return nil, fmt.Errorf("Invalid sha1sum, expected a 40-char hex representation of a hash")
	}

	// N.B. we don't do a lot of rigorous checking of arguments besides the sha1sum and id (those are essential to identify the part)

	pID := strings.TrimSpace(id)
	if id == "" {
		pID = strings.TrimSpace(sha1sum)
	}

	if part, exists := p.pkg.Parts[pID]; exists {
		return nil, fmt.Errorf("Provided pkg part id conflicts with already existing part. Existing: %v", part)
	}

	for _, part := range p.pkg.Parts {
		if part.Sha1sum == sha1sum {
			return nil, fmt.Errorf("Provided pkg part sha1sum conflicts with already existing part. Existing: %v")
		}
	}

	for id, dockerImageName := range p.pkg.Meta.Provides.Images {
		if id == pID {
			return nil, fmt.Errorf("Provided pkg part id conflicts with already existing entry in meta section. Existing: %v")
		}

		if dockerImageName == dockerImageRepoTag {
			return nil, fmt.Errorf("Provided pkg part's dockerImageRepoTag conflicts with already existing entry in meta section. Existing: %v")
		}
	}

	if len(signatures) == 0 && !p.permitEmptySignatures {
		return nil, errors.New("Provided signatures slice is empty and this builder is configured to disallow empty signatures for each part")
	}

	if len(sources) == 0 {
		return nil, errors.New("No provided sources")
	}

	part := DockerImagePart{
		Id:         pID,
		Sha1sum:    sha1sum,
		Signatures: signatures,
		Bytes:      bytes,
		Sources:    sources,
	}

	p.pkg.Parts[id] = part
	p.pkg.Meta.Provides.Images[id] = dockerImageRepoTag
	return p, nil
}

func (p *PkgBuilder) Build() *Pkg {
	return p.pkg
}

// PartsType is a faux-enum identifying the type of parts in this Pkg
type PartsType string

const (
	FILE PartsType = "FILE"
)

// ProvidesType is a faux-enum identifying the type that this Pkg's parts
// provide.
type ProvidesType string

const (
	DOCKER ProvidesType = "DOCKER"
)

// DockerImagePartName is a mapping b/n a "part" name (see the DockerImagePart type) and its docker image name
type DockerImagePartNames map[string]string

// DockerPartsProvides is one (and the only, for now) PartsProvides type. This
// is a metadata structure that describes the use of the Parts in a Pkg.
type DockerPartsProvides struct {
	ProvidesType ProvidesType         `json:"provides_types"`
	Images       DockerImagePartNames `json:"images"`
}

type Meta struct {
	PartsType   PartsType           `json:"parts_type"`
	Author      string              `json:"author"`
	SpecVersion string              `json:"spec_version"`
	Provides    DockerPartsProvides `json:"provides"`
}

type PartSource struct {
	URL string `json:"url"`
}

type DockerImagePart struct {
	Id         string       `json:"id"`
	Sha1sum    string       `json:"sha1sum"`
	Signatures []string     `json:"signatures"`
	Bytes      uint         `json:"bytes"`
	Sources    []PartSource `json:"sources"`
}
