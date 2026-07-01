package incident_test

import (
	"context"
	"fmt"
	"log"

	incident "github.com/incident-io/sdk-go"
)

// Create a client and list incidents.
func Example() {
	c, err := incident.New("my-api-key")
	if err != nil {
		log.Fatal(err)
	}

	resp, err := c.IncidentsV2ListWithResponse(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}
	if resp.JSON200 == nil {
		log.Fatalf("unexpected status %d: %s", resp.StatusCode(), resp.Body)
	}

	for _, inc := range resp.JSON200.Incidents {
		fmt.Printf("%s %s\n", inc.Reference, inc.Name)
	}
}

// Enable automatic retries of transient failures (off by default).
func ExampleWithRetries() {
	c, err := incident.New("my-api-key", incident.WithRetries())
	if err != nil {
		log.Fatal(err)
	}
	_ = c
}
