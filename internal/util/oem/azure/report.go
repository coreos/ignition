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
	"encoding/xml"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/ignition/internal/util/retry"
)

const (
	healthReportUriFormat = "http://%s/machine?comp=health"
	GoalStateUri          = "http://%s/machine/?comp=goalstate"

	AgentName             = "com.coreos.metadata"
	FabricProtocolVersion = "2012-11-30"

	LeaseRetryInterval = 500 * time.Millisecond
)

var (
	headers = map[string][]string{
		"x-ms-agent-name": {AgentName},
		"x-ms-version":    {FabricProtocolVersion},
		"Content-Type":    {"text/xml; charset=utf-8"},
	}
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

func getClient() retry.Client {
	client := retry.Client{
		InitialBackoff: time.Second,
		MaxBackoff:     time.Second * 5,
		MaxAttempts:    10,
		Header: map[string][]string{
			"x-ms-agent-name": {AgentName},
			"x-ms-version":    {FabricProtocolVersion},
			"Content-Type":    {"text/xml; charset=utf-8"},
		},
	}

	return client
}

func buildHealthReport(incarnation, container_id, role_instance_id, status, substatus, description string) ([]byte, error) {
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
		ContainerId:          container_id,
		InstanceId:           role_instance_id,
		State:                status,
		Details:              details,
	}, "  ", "    ")
	if err != nil {
		data = append([]byte(xml.Header), data...)
	}
	return data, err
}

func getGoalState(addr net.IP) (GoalState, error) {
	xmlBlob, err := getClient().Getf(GoalStateUri, addr.String())
	if err != nil {
		return GoalState{}, fmt.Errorf("failed to fetch goal state: %v", err)
	}
	var gs GoalState
	err = xml.Unmarshal(xmlBlob, &gs)
	if err != nil {
		return GoalState{}, err
	}
	return gs, nil
}

func findLease() (*os.File, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("could not list interfaces: %v", err)
	}

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
		time.Sleep(LeaseRetryInterval)
	}
}

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

func reportHealth(status, substatus, description string) error {
	fabricAddress, err := getFabricAddress()
	if err != nil {
		return err
	}
	goalState, err := getGoalState(fabricAddress)
	if err != nil {
		return err
	}
	if len(goalState.RoleInstanceList) == 0 {
		return fmt.Errorf("role instance list in goal state cannot be empty")
	}
	healthReport, err := buildHealthReport(goalState.Incarnation, goalState.ContainerId, goalState.RoleInstanceList[0].InstanceId, status, substatus, description)
	if err != nil {
		return err
	}
	_, err = getClient().Postf(healthReport, healthReportUriFormat, fabricAddress.String())
	if err != nil {
		return err
	}
	return nil
}

func ReportProvisioningStarting() error {
	return reportHealth("NotReady", "Provisioning", "Starting")
}

func ReportProvisioningFailed(description string) error {
	return reportHealth("NotReady", "ProvisioningFailed", description)
}

func ReportProvisioningSucceeded() error {
	return reportHealth("Ready", "", "")
}
