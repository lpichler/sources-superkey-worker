package main

import (
	"encoding/json"
	"fmt"

	l "github.com/redhatinsights/sources-superkey-worker/logger"
	"github.com/redhatinsights/sources-superkey-worker/provider"
	"github.com/redhatinsights/sources-superkey-worker/superkey"
	"github.com/segmentio/kafka-go"
)

// ProcessSuperkeyRequest - processes messages.
func ProcessSuperkeyRequest(msg kafka.Message) {
	eventType := getEventType(msg.Headers)

	switch eventType {
	case "create_application":
		l.Log.Info("Processing `create_application` request")

		req, err := parseSuperKeyRequest(msg.Value)
		if err != nil {
			l.Log.Warnf("Error parsing request: %v", err)
			return
		}

		fmt.Printf("%v\n", req)
		l.Log.Infof("Forging request: %v", req)
		newApp, err := provider.Forge(req)
		if err != nil {
			l.Log.Errorf("Error forging request %v, error: %v", req, err)
			l.Log.Errorf("Tearing down superkey request %v", req)

			errors := provider.TearDown(newApp)
			if len(errors) != 0 {
				for _, err := range errors {
					l.Log.Errorf("Error during teardown: %v", err)
				}
			}

			return
		}
		l.Log.Infof("Finished Forging request: %v", req)

		err = newApp.CreateInSourcesAPI()
		if err != nil {
			l.Log.Errorf("Failed to POST req to sources-api: %v, tearing down.", req)
			provider.TearDown(newApp)
		}

		l.Log.Infof("Finished processing `create_application` request for tenant %v type %v", req.TenantID, req.ApplicationType)

	case "delete_application":
		// TODO: teardown
		l.Log.Warn("delete_application not implemented yet")

	default:
		l.Log.Warn("Unknown event_type")
	}
}

// parseSuperKeyRequest - parses a kafka message's value ([]byte) into a Request struct
// returns: *Request
func parseSuperKeyRequest(value []byte) (*superkey.CreateRequest, error) {
	request := superkey.CreateRequest{}
	err := json.Unmarshal(value, &request)
	if err != nil {
		return nil, err
	}

	return &request, nil
}

// getEventType - iterates through headers to find the `event_type` header
// from miq-messaging.
// returns: event_type value
func getEventType(headers []kafka.Header) string {
	for _, header := range headers {
		if header.Key == "event_type" {
			return string(header.Value)
		}
	}

	return ""
}
