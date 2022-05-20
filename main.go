package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("run error: %s", err)
	}
}

func run() error {
	mc := machineClient{
		client: gcpClient{},
	}

	// normal downscale delete: care about the result --> no error.
	if err := mc.deleteInstance(uuid.New()); err != nil {
		return fmt.Errorf("delete instance: %w", err)
	}
	// stuck agent intro secret delete case --> care about specific error.
	if err := mc.deleteInstance(uuid.Nil); err != nil {
		if !errors.Is(err, ErrGCPInstanceDelete) {
			return fmt.Errorf("delete instance: %w", err)
		}
		log.Printf("could not find vm to delete: %s\n", err)
	}
	return nil
}

// sentinel error defined in the gcp client wrapper code.
var ErrGCPInstanceDelete = errors.New("could not delete GCP instance")

// the code piece where we could wrap the actual GCP client call
func (g gcpClient) deleteGCPInstance(id uuid.UUID) error {
	if id == uuid.Nil {
		return ErrGCPInstanceDelete
	}
	return nil
}

type gcpClient struct{}

type machineClient struct {
	client gcpClient
}

// machine client code that calls into the GCP specific code we write.
func (m machineClient) deleteInstance(id uuid.UUID) error {
	err := m.client.deleteGCPInstance(id)
	if err != nil {
		return fmt.Errorf("delete gcp instance: %w", err)
	}
	return nil
}
