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
	machineClient := &machineClient{
		client: gcpClient{},
	}

	var mc machineClienter = machineClient

	strict := strictStrategy{mc: *machineClient}
	errorProne := errorProneStrategy{mc: *machineClient}
	// normal downscale delete: care about the result --> no error.
	mc.setStopPrebootedLinuxStrategy(strict)
	if err := mc.stopPrebootedLinuxGCP(uuid.New()); err != nil {
		return fmt.Errorf("delete instance: %w", err)
	}
	// stuck agent intro secret delete case --> care about specific error.
	mc.setStopPrebootedLinuxStrategy(errorProne)
	if err := mc.stopPrebootedLinuxGCP(uuid.Nil); err != nil {
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
func (g gcpClient) DeleteInstance(id uuid.UUID) error {
	if id == uuid.Nil {
		return ErrGCPInstanceDelete
	}
	return nil
}

// the wrapping code what we have around the real gcp client
type gcpClient struct{}

type machineClient struct {
	client            gcpClient
	stopLinuxStrategy strategier
}

type machineClienter interface {
	setStopPrebootedLinuxStrategy(strategy strategier)
	stopPrebootedLinuxGCP(id uuid.UUID) error
	withKillAgent(id uuid.UUID, callBack func(id uuid.UUID) error) error
}

func (m *machineClient) setStopPrebootedLinuxStrategy(concreteStrategy strategier) {
	m.stopLinuxStrategy = concreteStrategy
}

type strategier interface {
	stop(id uuid.UUID) error
}

type errorProneStrategy struct {
	mc machineClient
}

type strictStrategy struct {
	mc machineClient
}

func (s errorProneStrategy) stop(id uuid.UUID) error {
	err := s.mc.withKillAgent(id, func(uuid.UUID) error {
		err := s.mc.client.DeleteInstance(id)
		if err != nil && errors.Is(err, ErrGCPInstanceDelete) {
			log.Print("skip error during deleting GCE instance")
		} else if err != nil {
			return fmt.Errorf("delete instance: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("kill agent: %w", err)
	}
	return nil
}

func (s strictStrategy) stop(id uuid.UUID) error {
	err := s.mc.withKillAgent(id, func(uuid.UUID) error {
		if err := s.mc.client.DeleteInstance(id); err != nil {
			return fmt.Errorf("delete instance: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("kill agent: %w", err)
	}
	return nil
}

// machine client code that calls into the GCP specific code we write.
func (m *machineClient) stopPrebootedLinuxGCP(id uuid.UUID) error {
	if err := m.stopLinuxStrategy.stop(id); err != nil {
		return fmt.Errorf("delete gcp instance: %w", err)
	}
	return nil
}

func (mc *machineClient) withKillAgent(id uuid.UUID, callBack func(id uuid.UUID) error) error {
	err := callBack(id)
	if err != nil {
		fmt.Printf("kept agent (%s)\n", id)
		return fmt.Errorf("callback: %w", err)
	}
	fmt.Printf("removed agent (%s)\n", id)
	return nil
}
