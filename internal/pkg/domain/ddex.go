package domain

import (
	"context"
	"encoding/xml"
)

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

// ERNMessage represents a DDEX ERN message
type ERNMessage struct {
	XMLName       xml.Name      `xml:"ernMessage"`
	MessageHeader MessageHeader `xml:"messageHeader"`
	ResourceList  ResourceList  `xml:"resourceList"`
	ReleaseList   ReleaseList   `xml:"releaseList"`
	DealList      DealList      `xml:"dealList"`
}

// MessageHeader represents the DDEX message header
type MessageHeader struct {
	MessageID              string `xml:"messageId"`
	MessageSender          string `xml:"messageSender"`
	MessageRecipient       string `xml:"messageRecipient"`
	MessageCreatedDateTime string `xml:"messageCreatedDateTime"`
}

// ResourceList represents a list of resources
type ResourceList struct {
	SoundRecordings []SoundRecording `xml:"soundRecording"`
}

// ReleaseList represents a list of releases
type ReleaseList struct {
	Releases []Release `xml:"release"`
}

// DealList represents a list of deals
type DealList struct {
	ReleaseDeals []ReleaseDeal `xml:"releaseDeal"`
}

// SoundRecording represents a DDEX sound recording
type SoundRecording struct {
	ISRC               string           `xml:"isrc"`
	Title              Title            `xml:"title"`
	Duration           string           `xml:"duration"`
	TechnicalDetails   TechnicalDetails `xml:"technicalDetails"`
	SoundRecordingType string           `xml:"soundRecordingType"`
	ResourceReference  string           `xml:"resourceReference"`
}

// Title represents a DDEX title
type Title struct {
	TitleText string `xml:"titleText"`
}

// TechnicalDetails represents technical details of a recording
type TechnicalDetails struct {
	TechnicalResourceDetailsReference string `xml:"technicalResourceDetailsReference"`
	Audio                             Audio  `xml:"audio"`
}

// Audio represents audio technical details
type Audio struct {
	Format     string `xml:"audioFormat"`
	BitRate    int    `xml:"bitRate"`
	SampleRate int    `xml:"sampleRate"`
}

// Release represents a DDEX release
type Release struct {
	ReleaseID      ReleaseID `xml:"releaseId"`
	ReferenceTitle Title     `xml:"referenceTitle"`
	ReleaseType    string    `xml:"releaseType"`
}

// ReleaseID represents a DDEX release ID
type ReleaseID struct {
	ICPN string `xml:"icpn"`
}

// ReleaseDeal represents a DDEX release deal
type ReleaseDeal struct {
	DealReleaseReference string `xml:"dealReleaseReference"`
	Deal                 Deal   `xml:"deal"`
}

// Deal represents a DDEX deal
type Deal struct {
	Territory Territory `xml:"territory"`
	DealTerms DealTerms `xml:"dealTerms"`
}

// Territory represents a DDEX territory
type Territory struct {
	TerritoryCode string `xml:"territoryCode"`
}

// DealTerms represents DDEX deal terms
type DealTerms struct {
	CommercialModelType string `xml:"commercialModelType"`
	Usage               Usage  `xml:"usage"`
}

// Usage represents DDEX usage terms
type Usage struct {
	UseType string `xml:"useType"`
}
