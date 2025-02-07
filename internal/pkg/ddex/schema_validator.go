package ddex

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/beevik/etree"
)

// XMLSchemaValidator implements the SchemaValidator interface for XML schema validation
type XMLSchemaValidator struct {
	cachedDocs map[string]*etree.Document
}

// NewXMLSchemaValidator creates a new XMLSchemaValidator instance
func NewXMLSchemaValidator() *XMLSchemaValidator {
	return &XMLSchemaValidator{
		cachedDocs: make(map[string]*etree.Document),
	}
}

// ValidateAgainstSchema validates XML data against basic XML rules and structure
func (v *XMLSchemaValidator) ValidateAgainstSchema(xmlData []byte, schemaPath string) error {
	// First, validate that it's well-formed XML
	if err := xml.Unmarshal(xmlData, new(interface{})); err != nil {
		return fmt.Errorf("malformed XML: %w", err)
	}

	// Load and parse the XML document using etree
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(xmlData); err != nil {
		return fmt.Errorf("failed to parse XML document: %w", err)
	}

	// Basic structural validation
	root := doc.Root()
	if root == nil {
		return fmt.Errorf("XML document has no root element")
	}

	// Validate required elements based on DDEX ERN 4.3 schema
	if err := v.validateRequiredElements(root); err != nil {
		return fmt.Errorf("DDEX validation failed: %w", err)
	}

	return nil
}

// validateRequiredElements checks for required DDEX ERN elements
func (v *XMLSchemaValidator) validateRequiredElements(root *etree.Element) error {
	// Check root element name
	if root.Tag != "ernMessage" {
		return fmt.Errorf("root element must be 'ernMessage', got '%s'", root.Tag)
	}

	// Required elements
	required := []string{
		"messageHeader/messageId",
		"messageHeader/messageSender",
		"messageHeader/messageRecipient",
		"messageHeader/messageCreatedDateTime",
		"resourceList",
		"releaseList",
		"dealList",
	}

	for _, path := range required {
		elements := root.FindElements(path)
		if len(elements) == 0 {
			return fmt.Errorf("missing required element: %s", path)
		}
	}

	// Validate sound recordings
	soundRecordings := root.FindElements("resourceList/soundRecording")
	for i, sr := range soundRecordings {
		if err := v.validateSoundRecording(sr, i); err != nil {
			return err
		}
	}

	// Validate releases
	releases := root.FindElements("releaseList/release")
	for i, r := range releases {
		if err := v.validateRelease(r, i); err != nil {
			return err
		}
	}

	// Validate deals
	deals := root.FindElements("dealList/releaseDeal")
	for i, d := range deals {
		if err := v.validateDeal(d, i); err != nil {
			return err
		}
	}

	return nil
}

// validateSoundRecording validates a single sound recording element
func (v *XMLSchemaValidator) validateSoundRecording(sr *etree.Element, index int) error {
	required := []string{
		"isrc",
		"title/titleText",
		"duration",
		"technicalDetails/technicalResourceDetailsReference",
		"soundRecordingType",
		"resourceReference",
	}

	for _, path := range required {
		element := sr.FindElement(path)
		if element == nil {
			return fmt.Errorf("sound recording %d: missing required element: %s", index+1, path)
		}

		// Validate non-empty values
		if strings.TrimSpace(element.Text()) == "" {
			return fmt.Errorf("sound recording %d: empty value for required element: %s", index+1, path)
		}
	}

	return nil
}

// validateRelease validates a single release element
func (v *XMLSchemaValidator) validateRelease(r *etree.Element, index int) error {
	required := []string{
		"releaseId/icpn",
		"referenceTitle/titleText",
		"releaseType",
	}

	for _, path := range required {
		element := r.FindElement(path)
		if element == nil {
			return fmt.Errorf("release %d: missing required element: %s", index+1, path)
		}

		// Validate non-empty values
		if strings.TrimSpace(element.Text()) == "" {
			return fmt.Errorf("release %d: empty value for required element: %s", index+1, path)
		}
	}

	return nil
}

// validateDeal validates a single deal element
func (v *XMLSchemaValidator) validateDeal(d *etree.Element, index int) error {
	required := []string{
		"dealReleaseReference",
		"deal/territory/territoryCode",
		"deal/dealTerms/commercialModelType",
		"deal/dealTerms/usage/useType",
	}

	for _, path := range required {
		element := d.FindElement(path)
		if element == nil {
			return fmt.Errorf("deal %d: missing required element: %s", index+1, path)
		}

		// Validate non-empty values
		if strings.TrimSpace(element.Text()) == "" {
			return fmt.Errorf("deal %d: empty value for required element: %s", index+1, path)
		}
	}

	return nil
}

// ClearCache clears the document cache
func (v *XMLSchemaValidator) ClearCache() {
	v.cachedDocs = make(map[string]*etree.Document)
}
