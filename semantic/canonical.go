package semantic

import "time"

// Canonical semantic types following Schema.org hierarchy
// Uses composition + Type field + Properties map for maximum flexibility

// SemanticThing is the base type for ALL entities
// Replaces: SemanticAgent, SemanticInstrument, SemanticError
// Covers: Person, Organization, Place, Product, Event, Service, Error, etc.
type SemanticThing struct {
	Context     string                 `json:"@context,omitempty"`
	Type        string                 `json:"@type"`
	ID          string                 `json:"@id,omitempty"`
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	URL         string                 `json:"url,omitempty"`
	Image       string                 `json:"image,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty"` // Additional fields
}

// SemanticCreativeWork represents content, media, documents, code, datasets
// Replaces: SemanticObject (partially), SemanticMediaObject, SemanticDigitalDocument
// Covers: Article, Book, Dataset, Image, Video, Audio, SoftwareSourceCode, WebPage, Message
type SemanticCreativeWork struct {
	SemanticThing                      // Embedded base
	Text                string         `json:"text,omitempty"`
	ContentURL          string         `json:"contentUrl,omitempty"`
	EncodingFormat      string         `json:"encodingFormat,omitempty"`
	ContentSize         int64          `json:"contentSize,omitempty"`
	DateCreated         *time.Time     `json:"dateCreated,omitempty"`
	DateModified        *time.Time     `json:"dateModified,omitempty"`
	Author              *SemanticThing `json:"author,omitempty"`
	About               interface{}    `json:"about,omitempty"`
	Keywords            []string       `json:"keywords,omitempty"`
	ProgrammingLanguage string         `json:"programmingLanguage,omitempty"`
	CodeRepository      string         `json:"codeRepository,omitempty"`
	RuntimePlatform     string         `json:"runtimePlatform,omitempty"`
}

// SemanticEntryPoint represents API endpoints, URLs, interfaces for accessing actions/services
type SemanticEntryPoint struct {
	Type           string            `json:"@type"`
	URLTemplate    string            `json:"urlTemplate,omitempty"`
	HTTPMethod     string            `json:"httpMethod,omitempty"`
	ContentType    string            `json:"encodingType,omitempty"`
	Headers        map[string]string `json:"headers,omitempty"`
	ActionPlatform []string          `json:"actionPlatform,omitempty"`
}

// CanonicalSemanticAction is the enhanced action type using canonical composition
// Prefer this over SemanticAction for new code
type CanonicalSemanticAction struct {
	Context      string `json:"@context"`
	Type         string `json:"@type"` // SearchAction, CreateAction, UpdateAction, DeleteAction, etc.
	ID           string `json:"@id,omitempty"`
	Name         string `json:"name,omitempty"`
	Description  string `json:"description,omitempty"`
	ActionStatus string `json:"actionStatus,omitempty"` // PotentialActionStatus, ActiveActionStatus, CompletedActionStatus, FailedActionStatus

	// WHO and HOW (now use SemanticThing instead of specific types)
	Agent      *SemanticThing `json:"agent,omitempty"`      // Who performs (Person, Organization, SoftwareApplication)
	Instrument *SemanticThing `json:"instrument,omitempty"` // Tool used (SoftwareApplication)

	// WHAT (now more flexible - can be Thing or CreativeWork)
	Object interface{}         `json:"object,omitempty"`
	Target *SemanticEntryPoint `json:"target,omitempty"`

	// RESULT/STATE (now use SemanticThing for errors)
	Result interface{}    `json:"result,omitempty"`
	Error  *SemanticThing `json:"error,omitempty"`

	// TIMING
	StartTime *time.Time `json:"startTime,omitempty"`
	EndTime   *time.Time `json:"endTime,omitempty"`

	// SCHEDULING
	Schedule *SemanticSchedule `json:"schedule,omitempty"`

	// DEPENDENCIES
	Requires []string `json:"requires,omitempty"` // @id references

	// EXECUTION CONFIG (non-semantic, stored in properties)
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// Helper constructors for common patterns

// NewPerson creates a SemanticThing representing a person
func NewPerson(name, email string) *SemanticThing {
	return &SemanticThing{
		Type: "Person",
		Name: name,
		Properties: map[string]interface{}{
			"email": email,
		},
	}
}

// NewSoftwareApplication creates a SemanticThing representing a software tool
func NewSoftwareApplication(name string) *SemanticThing {
	return &SemanticThing{
		Type: "SoftwareApplication",
		Name: name,
	}
}

// NewOrganization creates a SemanticThing representing an organization
func NewOrganization(name, url string) *SemanticThing {
	return &SemanticThing{
		Type: "Organization",
		Name: name,
		URL:  url,
	}
}

// NewError creates a SemanticThing representing an error
func NewError(message string) *SemanticThing {
	return &SemanticThing{
		Type: "Error",
		Name: message,
	}
}

// NewDataset creates a SemanticCreativeWork representing a dataset
func NewDataset(name, contentURL, encodingFormat string) *SemanticCreativeWork {
	return &SemanticCreativeWork{
		SemanticThing: SemanticThing{
			Type: "Dataset",
			Name: name,
		},
		ContentURL:     contentURL,
		EncodingFormat: encodingFormat,
	}
}

// NewSoftwareSourceCode creates a SemanticCreativeWork representing code
func NewSoftwareSourceCode(name, codeRepo, language string) *SemanticCreativeWork {
	return &SemanticCreativeWork{
		SemanticThing: SemanticThing{
			Type: "SoftwareSourceCode",
			Name: name,
		},
		CodeRepository:      codeRepo,
		ProgrammingLanguage: language,
	}
}

// NewHTTPEntryPoint creates a SemanticEntryPoint for HTTP/API calls
func NewHTTPEntryPoint(url, method string, headers map[string]string) *SemanticEntryPoint {
	return &SemanticEntryPoint{
		Type:        "EntryPoint",
		URLTemplate: url,
		HTTPMethod:  method,
		Headers:     headers,
	}
}
