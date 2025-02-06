package domain

import "context"

// DDEX format version
const (
	DDEXERN43 = "4.3"
)

// DDEXService handles DDEX format operations
type DDEXService interface {
	// ValidateTrack validates a track's metadata against DDEX schema
	ValidateTrack(ctx context.Context, track *Track) (bool, []string)

	// ExportTrack exports a single track to DDEX format
	ExportTrack(ctx context.Context, track *Track) (string, error)

	// ExportTracks exports multiple tracks to DDEX format
	ExportTracks(ctx context.Context, tracks []*Track) (string, error)
}

// DDEXERN43Message represents a DDEX ERN 4.3 message
type DDEXERN43Message struct {
	XMLName                string        `xml:"ern:NewReleaseMessage"`
	XMLNs                  string        `xml:"xmlns,attr"`
	XMLNsErn               string        `xml:"xmlns:ern,attr"`
	MessageSchemaVersionId string        `xml:"MessageSchemaVersionId"`
	MessageHeader          MessageHeader `xml:"MessageHeader"`
	ResourceList           ResourceList  `xml:"ResourceList"`
	ReleaseList            ReleaseList   `xml:"ReleaseList"`
	DealList               DealList      `xml:"DealList"`
}

// MessageHeader represents the DDEX message header
type MessageHeader struct {
	MessageID          string `xml:"MessageId"`
	MessageSender      string `xml:"MessageSender"`
	MessageRecipient   string `xml:"MessageRecipient"`
	MessageCreatedDate string `xml:"MessageCreatedDateTime"`
	MessageControlType string `xml:"MessageControlType"`
}

// ResourceList contains all resources in the message
type ResourceList struct {
	SoundRecordings []SoundRecording `xml:"SoundRecording"`
}

// ReleaseList contains all releases in the message
type ReleaseList struct {
	Releases []Release `xml:"Release"`
}

// DealList contains all deals in the message
type DealList struct {
	ReleaseDeals []ReleaseDeal `xml:"ReleaseDeal"`
}

// ResourceId represents a DDEX resource identifier
type ResourceId struct {
	ISRC string `xml:"ISRC"`
	Type string `xml:"Type,omitempty"`
}

// Title represents a DDEX title
type Title struct {
	TitleText string `xml:"TitleText"`
	Type      string `xml:"Type,omitempty"`
}

// Audio represents DDEX audio technical details
type Audio struct {
	Format     string `xml:"AudioCodec"`
	BitRate    int    `xml:"BitRate"`
	SampleRate int    `xml:"SampleRate"`
}

// TechnicalDetails represents DDEX technical metadata
type TechnicalDetails struct {
	Audio                             Audio  `xml:"Audio"`
	TechnicalResourceDetailsReference string `xml:"TechnicalResourceDetailsReference"`
}

// SoundRecording represents a DDEX sound recording
type SoundRecording struct {
	ResourceId         ResourceId       `xml:"SoundRecordingId"`
	Title              Title            `xml:"ReferenceTitle"`
	Duration           string           `xml:"Duration"`
	TechnicalDetails   TechnicalDetails `xml:"TechnicalDetails"`
	SoundRecordingType string           `xml:"SoundRecordingType"`
	ResourceReference  string           `xml:"ResourceReference"`
}

// Release represents a DDEX release
type Release struct {
	ReleaseId      ReleaseId `xml:"ReleaseId"`
	ReferenceTitle Title     `xml:"ReferenceTitle"`
	ReleaseType    string    `xml:"ReleaseType"`
}

// ReleaseId represents a DDEX release identifier
type ReleaseId struct {
	ICPN string `xml:"ICPN"`
}

// ReleaseDeal represents a DDEX release deal
type ReleaseDeal struct {
	DealReleaseReference string `xml:"DealReleaseReference"`
	Deal                 Deal   `xml:"Deal"`
}

// Deal represents a DDEX deal
type Deal struct {
	Territory Territory `xml:"Territory"`
	DealTerms DealTerms `xml:"DealTerms"`
}

// Territory represents a DDEX territory
type Territory struct {
	TerritoryCode string `xml:"TerritoryCode"`
}

// DealTerms represents DDEX deal terms
type DealTerms struct {
	CommercialModelType string `xml:"CommercialModelType"`
	Usage               Usage  `xml:"Usage"`
}

// Usage represents DDEX usage terms
type Usage struct {
	UseType string `xml:"UseType"`
}
