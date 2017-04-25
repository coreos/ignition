// Copyright 2017 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package azure

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/ignition/config/types"
	"github.com/coreos/ignition/internal/log"
	"github.com/coreos/ignition/internal/resource"
)

const (
	healthReportUriFormat = "http://%s/machine?comp=health"
	goalStateUri          = "http://%s/machine/?comp=goalstate"

	agentName             = "com.coreos.metadata"
	fabricProtocolVersion = "2012-11-30"

	leaseRetryInterval = 500 * time.Millisecond
)

type HealthReport struct {
	XMLName              xml.Name       `xml:"Health"`
	Xsi                  string         `xml:"xmlns:xsi,attr"`
	Xsd                  string         `xml:"xmlns:xsd,attr"`
	GoalStateIncarnation string         `xml:"GoalStateIncarnation"`
	ContainerId          string         `xml:"Container>ContainerId"`
	InstanceId           string         `xml:"Container>RoleInstanceList>Role>InstanceId"`
	State                string         `xml:"Container>RoleInstanceList>Role>Health>State"`
	Details              *HealthDetails `xml:"Container>RoleInstanceList>Role>Health>Details"`
}

type HealthDetails struct {
	SubStatus   string `xml:"SubStatus"`
	Description string `xml:"Description"`
}

type GoalState struct {
	XMLName               xml.Name                `xml:"GoalState"`
	Version               string                  `xml:"Version"`
	Incarnation           string                  `xml:"Incarnation"`
	ExpectedState         string                  `xml:"Machine>ExpectedState"`
	StopRolesDeadlineHint string                  `xml:"Machine>StopRolesDeadlineHint"`
	LBProbePorts          []int                   `xml:"Machine>LBProbePorts>Port"`
	ExpectHealthReport    string                  `xml:"Machine>ExpectHealthReport"`
	ContainerId           string                  `xml:"Container>ContainerId"`
	RoleInstanceList      []GoalStateRoleInstance `xml:"Container>RoleInstanceList>RoleInstance"`
}

type GoalStateRoleInstance struct {
	XMLName                  xml.Name `xml:"RoleInstance"`
	InstanceId               string   `xml:"InstanceId"`
	State                    string   `xml:"State"`
	HostingEnvironmentConfig string   `xml:"Configuration>HostingEnvironmentConfig"`
	SharedConfig             string   `xml:"Configuration>SharedConfig"`
	ExtensionsConfig         string   `xml:"Configuration>ExtensionsConfig"`
	FullConfig               string   `xml:"Configuration>FullConfig"`
	Certificates             string   `xml:"Configuration>Certificates"`
	ConfigName               string   `xml:"Configuration>ConfigName"`
}

type AzureProvisioner struct {
	client  resource.HttpClient
	headers map[string][]string
}

func NewAzureProvisioner(l *log.Logger) *AzureProvisioner {
	return &AzureProvisioner{
		client: resource.NewHttpClient(l, types.Timeouts{}, 400),
		headers: map[string][]string{
			"x-ms-agent-name": {agentName},
			"x-ms-version":    {fabricProtocolVersion},
			"Content-Type":    {"text/xml; charset=utf-8"},
		},
	}
}

// buildHealthReport will convert the goal state and the desired message into an
// XML-encoded health report message
func buildHealthReport(incarnation, containerId, roleInstanceId, status, substatus, description string) ([]byte, error) {
	var details *HealthDetails
	if substatus != "" {
		details = &HealthDetails{
			SubStatus:   substatus,
			Description: description,
		}
	}
	data, err := xml.MarshalIndent(HealthReport{
		Xsi:                  "http://www.w3.org/2001/XMLSchema-instance",
		Xsd:                  "http://www.w3.org/2001/XMLSchema",
		GoalStateIncarnation: incarnation,
		ContainerId:          containerId,
		InstanceId:           roleInstanceId,
		State:                status,
		Details:              details,
	}, "  ", "    ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), data...), err
}

// getGoalState will send a request to the given IP address for the XML-encoded
// goal state of this machine, and then decodes the message and returns it.
func (a *AzureProvisioner) getGoalState(addr net.IP) (GoalState, error) {
	xmlBlobReader, _, err := a.client.Get(fmt.Sprintf(goalStateUri, addr.String()), a.headers)
	if err != nil {
		return GoalState{}, fmt.Errorf("failed to fetch goal state: %v", err)
	}
	defer xmlBlobReader.Close()
	xmlBlob, err := ioutil.ReadAll(xmlBlobReader)
	if err != nil {
		return GoalState{}, fmt.Errorf("failed to read goal state: %v", err)
	}
	var gs GoalState
	err = xml.Unmarshal(xmlBlob, &gs)
	if err != nil {
		return GoalState{}, err
	}
	return gs, nil
}

// findLease will attempt to find a systemd lease file for one of the interfaces
// enumerated by net.Interfaces()
func findLease() (*os.File, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("could not list interfaces: %v", err)
	}

	// It will take an unknown amount of time for an interface to successfully
	// go through the DHCP dance, so keep retrying (and sleeping between tries)
	// indefinitely.
	for {
		for _, iface := range ifaces {
			lease, err := os.Open(fmt.Sprintf("/run/systemd/netif/leases/%d", iface.Index))
			if os.IsNotExist(err) {
				continue
			} else if err != nil {
				return nil, err
			} else {
				return lease, nil
			}
		}

		fmt.Printf("No leases found. Waiting...")
		time.Sleep(leaseRetryInterval)
	}
}

// getFabricAddress attempts to look up the address of the Azure fabric server
// (which I guess shares an IP address with the wire server?) by searching
// through information about the current DHCP lease for an OPTION_245 line.
func getFabricAddress() (net.IP, error) {
	lease, err := findLease()
	if err != nil {
		return nil, err
	}
	defer lease.Close()

	var rawEndpoint string
	line := bufio.NewScanner(lease)
	for line.Scan() {
		parts := strings.Split(line.Text(), "=")
		if parts[0] == "OPTION_245" && len(parts) == 2 {
			rawEndpoint = parts[1]
			break
		}
	}

	if len(rawEndpoint) == 0 || len(rawEndpoint) != 8 {
		return nil, fmt.Errorf("fabric endpoint not found in leases")
	}

	octets := make([]byte, 4)
	for i := 0; i < 4; i++ {
		octet, err := strconv.ParseUint(rawEndpoint[2*i:2*i+2], 16, 8)
		if err != nil {
			return nil, err
		}
		octets[i] = byte(octet)
	}

	return net.IPv4(octets[0], octets[1], octets[2], octets[3]), nil
}

// reportHealth sends a message to the Azure wire server, with the provided
// status, substatus, and description.
func (a *AzureProvisioner) reportHealth(status, substatus, description string) error {
	// Get the address of the server we need to talk to
	fabricAddress, err := getFabricAddress()
	if err != nil {
		return err
	}
	// Get the goal state via get request to the server we just looked up
	goalState, err := a.getGoalState(fabricAddress)
	if err != nil {
		return err
	}
	// There must be at least one item in the RoleInstanceList (otherwise this
	// instance doesn't exist)
	if len(goalState.RoleInstanceList) == 0 {
		return fmt.Errorf("role instance list in goal state cannot be empty")
	}
	// Build a health report with the goal state we fetched and the desired
	// message we wish to send
	healthReport, err := buildHealthReport(goalState.Incarnation, goalState.ContainerId, goalState.RoleInstanceList[0].InstanceId, status, substatus, description)
	if err != nil {
		return err
	}
	// Send the message to the wire server
	body, _, err := a.client.Post(fmt.Sprintf(healthReportUriFormat, fabricAddress.String()), a.headers, bytes.NewBuffer(healthReport))
	if err != nil {
		return err
	}
	body.Close()
	return nil
}

// ReportProvisioningStarting will send a message to the Azure wire server
// reporting that this machine is online but not ready, and provisioning has
// begun. This should be called once when provisioning begins (so the start of
// the disks stage)
func (a *AzureProvisioner) ReportProvisioningStarting() error {
	return a.reportHealth("NotReady", "Provisioning", "Starting")
}

// ReportProvisioningFailed will send a message to the Azure wire server
// reporting that provisioning has failed on this machine. This should be called
// if a failure occurs and the machine is not going to successfully finish
// booting.
func (a *AzureProvisioner) ReportProvisioningFailed(description string) error {
	return a.reportHealth("NotReady", "ProvisioningFailed", description)
}

// ReportProvisioningSucceeded will send a message to the Azure wire server
// reporting that provisioning has succeeded on this machine. This should be
// called once when provisiong has finished (so at the end of the files stage).
// If this is not called, Azure will never mark a machine as "ready".
func (a *AzureProvisioner) ReportProvisioningSucceeded() error {
	return a.reportHealth("Ready", "", "")
}
