package domain

import "encoding/xml"

// DDEXService defines the interface for DDEX operations
type DDEXService interface {
	ValidateMetadata(track *Track) (bool, []string, error)
	ExportERN(track *Track) ([]byte, error)
	BatchExportERN(tracks []*Track) ([]byte, error)
}

// DDEXERN43 represents the DDEX ERN 4.3 format structure
type DDEXERN43 struct {
	XMLName                xml.Name `xml:"ern:NewReleaseMessage"`
	MessageHeader          MessageHeader
	ResourceList           ResourceList
	ReleaseList            ReleaseList
	DealList               DealList
	XMLNs                  string `xml:"xmlns,attr"`
	XMLNsErn               string `xml:"xmlns:ern,attr"`
	MessageSchemaVersionId string `xml:"MessageSchemaVersionId,attr"`
}

// MessageHeader contains DDEX message metadata
type MessageHeader struct {
	MessageId              string `xml:"MessageId"`
	MessageSender          string `xml:"MessageSender"`
	MessageRecipient       string `xml:"MessageRecipient"`
	MessageCreatedDateTime string `xml:"MessageCreatedDateTime"`
	MessageControlType     string `xml:"MessageControlType"`
}

// ResourceList contains sound recording details
type ResourceList struct {
	SoundRecording []SoundRecording `xml:"SoundRecording"`
}

// SoundRecording represents a single audio track
type SoundRecording struct {
	SoundRecordingType  string           `xml:"SoundRecordingType"`
	SoundRecordingId    ResourceId       `xml:"SoundRecordingId"`
	ResourceReference   string           `xml:"ResourceReference"`
	ReferenceTitle      Title            `xml:"ReferenceTitle"`
	Duration            string           `xml:"Duration"`
	ParentalWarningType string           `xml:"ParentalWarningType"`
	TechnicalDetails    TechnicalDetails `xml:"TechnicalDetails"`
}

// ResourceId contains identifiers like ISRC
type ResourceId struct {
	ISRC string `xml:"ISRC"`
}

// Title contains track title information
type Title struct {
	TitleText string `xml:"TitleText"`
}

// TechnicalDetails contains audio specifications
type TechnicalDetails struct {
	TechnicalResourceDetailsReference string `xml:"TechnicalResourceDetailsReference"`
	Audio                             Audio  `xml:"Audio"`
}

// Audio contains audio-specific metadata
type Audio struct {
	AudioCodec string `xml:"AudioCodec"`
	Bitrate    int    `xml:"Bitrate"`
}

// ReleaseList contains release information
type ReleaseList struct {
	Release []Release `xml:"Release"`
}

// Release represents a music release
type Release struct {
	ReleaseId      ReleaseId `xml:"ReleaseId"`
	ReferenceTitle Title     `xml:"ReferenceTitle"`
	ReleaseType    string    `xml:"ReleaseType"`
}

// ReleaseId contains release identifiers
type ReleaseId struct {
	ICPN string `xml:"ICPN"`
}

// DealList contains licensing and territory information
type DealList struct {
	ReleaseDeal []ReleaseDeal `xml:"ReleaseDeal"`
}

// ReleaseDeal represents licensing terms
type ReleaseDeal struct {
	DealReleaseReference string `xml:"DealReleaseReference"`
	Deal                 Deal   `xml:"Deal"`
}

// Deal contains territory and usage terms
type Deal struct {
	Territory Territory `xml:"Territory"`
	DealTerms DealTerms `xml:"DealTerms"`
}

// Territory specifies geographical regions
type Territory struct {
	TerritoryCode string `xml:"TerritoryCode"`
}

// DealTerms contains usage rights and restrictions
type DealTerms struct {
	CommercialModelType string `xml:"CommercialModelType"`
	Usage               Usage  `xml:"Usage"`
}

// Usage defines how content can be used
type Usage struct {
	UseType string `xml:"UseType"`
}
