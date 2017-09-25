package horizonpkg

import (
	"crypto/sha1"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"io"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	// constant for now
	specVersion = "0.1.0"
)

type DockerImageParts map[string]DockerImagePart

type Pkg struct {
	ID    string           `json:"id"`
	Meta  *Meta            `json:"meta"`
	Parts DockerImageParts `json:"parts"`
}

func (p *Pkg) Serialize() ([]byte, error) {
	serial, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}

	return serial, nil
}

// imageIDs contains all intended parts' names; the builder will error if not all IDs fulfilled with parts.
// This is a check on the build process; the imageIDs are also a part of the immutable identity of the pkg.
type PkgBuilder struct {
	pkg                   *Pkg
	permitEmptySignatures bool
	imageIDs              []string
	partMutex             sync.Mutex
}

// creates an ID for the package that is repeatably calculable from the content
// TODO: provide functions to calculate the package ID from a pkg file.
func pkgID(author string, createTS int64, imageIDs []string) string {
	hash := sha1.New()

	io.WriteString(hash, author)

	tsBin := make([]byte, 8)
	binary.LittleEndian.PutUint64(tsBin, uint64(createTS))
	hash.Write(tsBin)

	sort.Strings(imageIDs)
	for _, id := range imageIDs {
		io.WriteString(hash, id)
	}

	return fmt.Sprintf("%x", hash.Sum(nil))
}

// NewDockerImagePkgBuilder is a factory method for a pkg builder. It's
// expected that after getting a reference to this type, one will use AddPart()
// and related functions to populate it and then call BuildPkg() to produce an
// immutable package.
func NewDockerImagePkgBuilder(partsType PartsType, author string, imageIDs []string) (*PkgBuilder, error) {

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

	createTS := time.Now().UnixNano()

	return &PkgBuilder{
		pkg: &Pkg{
			ID: pkgID(author, createTS, imageIDs),
			Meta: &Meta{
				PartsType:   partsType,
				Author:      author,
				SpecVersion: specVersion,
				CreateTS:    createTS,
				Provides:    provides,
			},
			Parts: DockerImageParts{},
		},
		permitEmptySignatures: false,
		partMutex:             sync.Mutex{},
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
// that the id is not required; if it is an empty string, the sha256sum will be
// used as the id instead. A valid sha256sum is a hex representation of the
// 256-bit sequence.
func (p *PkgBuilder) AddPart(id string, sha256sum string, dockerImageRepoTag string, signatures []string, bytes int64, sources ...PartSource) (*PkgBuilder, error) {

	if sha256sumInvalid, err := regexp.MatchString("[^0-9A-Za-z]", sha256sum); err != nil || sha256sumInvalid || len(sha256sum) != 64 {
		return nil, fmt.Errorf("Invalid sha256sum, expected a 64-char hex representation of a hash. Hash was %v chars in length", len(sha256sum))
	}

	// N.B. we don't do a lot of rigorous checking of arguments besides the sha256sum and id (those are essential to identify the part)

	pID := strings.TrimSpace(id)
	if id == "" {
		pID = strings.TrimSpace(sha256sum)
	}

	p.partMutex.Lock()
	idPartCheck, exists := p.pkg.Parts[pID]
	p.partMutex.Unlock()

	if exists {
		return nil, fmt.Errorf("Provided pkg part id conflicts with already existing part. Existing: %v", idPartCheck)
	}

	checkErr := false
	p.partMutex.Lock()
	for _, partCheck := range p.pkg.Parts {
		if partCheck.Sha256sum == sha256sum {
			checkErr = true
		}
	}
	p.partMutex.Unlock()
	if checkErr {
		return nil, fmt.Errorf("Provided pkg part sha256sum conflicts with already existing part. Existing: %v", sha256sum)
	}

	imageIDConflictErr := false
	imageRepoTagConflictErr := false
	p.partMutex.Lock()
	for id, dockerImageName := range p.pkg.Meta.Provides.Images {
		if id == pID {
			imageIDConflictErr = true
		}

		if dockerImageName == dockerImageRepoTag {
			imageRepoTagConflictErr = true
		}
	}
	p.partMutex.Unlock()
	if imageIDConflictErr {
		return nil, fmt.Errorf("Provided pkg part id conflicts with already existing entry in meta section. Existing: %v", pID)
	}

	if imageRepoTagConflictErr {
		return nil, fmt.Errorf("Provided pkg part's dockerImageRepoTag conflicts with already existing entry in meta section. Existing: %v", dockerImageRepoTag)
	}

	if len(signatures) == 0 && !p.permitEmptySignatures {
		return nil, errors.New("Provided signatures slice is empty and this builder is configured to disallow empty signatures for each part")
	}

	if len(sources) == 0 {
		return nil, errors.New("No provided sources")
	}

	part := DockerImagePart{
		Id:         pID,
		Sha256sum:  sha256sum,
		Signatures: signatures,
		Bytes:      bytes,
		Sources:    sources,
	}

	p.pkg.Parts[id] = part
	p.pkg.Meta.Provides.Images[id] = dockerImageRepoTag

	return p, nil
}

func (p *PkgBuilder) ID() string {
	return p.pkg.ID
}

func (p *PkgBuilder) Build() (*Pkg, []byte, error) {

	// check that all of the builder's images are in the package parts or error
	for _, imageID := range p.imageIDs {
		if _, ok := p.pkg.Parts[imageID]; !ok {
			return nil, nil, fmt.Errorf("Expected image with id: %v not in Package parts. Use *Pkg.AddPart() to add the appropriate part for this image ID.", imageID)
		}
	}

	serialized, err := p.pkg.Serialize()
	if err != nil {
		return nil, nil, err
	}

	return p.pkg, serialized, nil
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
	CreateTS    int64               `json:"createTS"` // unix nanoseconds
}

type PartSource struct {
	URL string `json:"url"`
}

type DockerImagePart struct {
	Id         string       `json:"id"`
	Sha256sum  string       `json:"sha256sum"`
	Signatures []string     `json:"signatures"`
	Bytes      int64        `json:"bytes"`
	Sources    []PartSource `json:"sources"`
}
