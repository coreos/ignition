// Copyright 2020 Red Hat, Inc.
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

package fixtures

// This file contains all the large constants used for testing
// the tls flow.

var (
	// PrivateKey is generated via:
	// openssl ecparam -genkey -name secp384r1 -out server.key
	PrivateKey = []byte(`-----BEGIN EC PARAMETERS-----
BgUrgQQAIg==
-----END EC PARAMETERS-----
-----BEGIN EC PRIVATE KEY-----
MIGkAgEBBDB6yW6RIYfTXdYVuPY0V0L6EtZ6vZD86vgbsw52Y3/U5nZ2JE++JrKu
tt2Xt/NMzG6gBwYFK4EEACKhZANiAAQDEhfHEulYKlANw9eR5l455gwzAIQuraa0
49RhvM7PPywaiD8DobteQmE8wn7cJSzOYw6GLvrL4Q1BO5EFUXknkW50t8lfnUeH
veCNsqvm82F1NVevVoExAUhDYmMREa4=
-----END EC PRIVATE KEY-----`)

	// PublicKey is generated via:
	// generate csr:
	// openssl req -new -key server.key -out server.csr
	// generate certificate:
	// openssl x509 -req -days 3650 -in server.csr -signkey server.key -out
	// server.crt -extensions v3_req -extfile extfile.conf
	// where extfile.conf has the following details:
	// $ cat extfile.conf
	// [ v3_req ]
	// subjectAltName = IP:127.0.0.1
	// subjectKeyIdentifier=hash
	// authorityKeyIdentifier=keyid
	// basicConstraints = critical,CA:TRUE
	PublicKey = []byte(`-----BEGIN CERTIFICATE-----
MIICzTCCAlKgAwIBAgIJALTP0pfNBMzGMAoGCCqGSM49BAMCMIGZMQswCQYDVQQG
EwJVUzETMBEGA1UECAwKQ2FsaWZvcm5pYTEWMBQGA1UEBwwNU2FuIEZyYW5jaXNj
bzETMBEGA1UECgwKQ29yZU9TIEluYzEUMBIGA1UECwwLRW5naW5lZXJpbmcxEzAR
BgNVBAMMCmNvcmVvcy5jb20xHTAbBgkqhkiG9w0BCQEWDm9lbUBjb3Jlb3MuY29t
MB4XDTE4MDEyNTAwMDczOVoXDTI4MDEyMzAwMDczOVowgZkxCzAJBgNVBAYTAlVT
MRMwEQYDVQQIDApDYWxpZm9ybmlhMRYwFAYDVQQHDA1TYW4gRnJhbmNpc2NvMRMw
EQYDVQQKDApDb3JlT1MgSW5jMRQwEgYDVQQLDAtFbmdpbmVlcmluZzETMBEGA1UE
AwwKY29yZW9zLmNvbTEdMBsGCSqGSIb3DQEJARYOb2VtQGNvcmVvcy5jb20wdjAQ
BgcqhkjOPQIBBgUrgQQAIgNiAAQDEhfHEulYKlANw9eR5l455gwzAIQuraa049Rh
vM7PPywaiD8DobteQmE8wn7cJSzOYw6GLvrL4Q1BO5EFUXknkW50t8lfnUeHveCN
sqvm82F1NVevVoExAUhDYmMREa6jZDBiMA8GA1UdEQQIMAaHBH8AAAEwHQYDVR0O
BBYEFEbFy0SPiF1YXt+9T3Jig2rNmBtpMB8GA1UdIwQYMBaAFEbFy0SPiF1YXt+9
T3Jig2rNmBtpMA8GA1UdEwEB/wQFMAMBAf8wCgYIKoZIzj0EAwIDaQAwZgIxAOul
t3MhI02IONjTDusl2YuCxMgpy2uy0MPkEGUHnUOsxmPSG0gEBCNHyeKVeTaPUwIx
AKbyaAqbChEy9CvDgyv6qxTYU+eeBImLKS3PH2uW5etc/69V/sDojqpH3hEffsOt
9g==
-----END CERTIFICATE-----`)

	// PrivateKey2 is generated via
	// openssl ecparam -genkey -name secp384r1 -out server.key
	PrivateKey2 = []byte(`-----BEGIN EC PARAMETERS-----
BgUrgQQAIg==
-----END EC PARAMETERS-----
-----BEGIN EC PRIVATE KEY-----
MIGkAgEBBDCfXncsl/kqihUWRHThBdGEDpv/bavwHYEi2tjrHiRkm+b7zhFlup8o
aP1l1zP1LhKgBwYFK4EEACKhZANiAAQ/J0D0C3h2a55JU3/EANe1d3e2/mfcoXGq
P8soiFdYntRIC4+V4dnRJuHRR+FHR/3531EIf2WXsoIJr/IRhR/j0tAeXpZ++G+E
vaooXf7gShnhRYKM4viPx4+DhSPjmqw=
-----END EC PRIVATE KEY-----`)

	// PublicKey2 is generate via:
	// generate csr:
	// openssl req -new -key server.key -out server.csr
	// generate certificate:
	// openssl x509 -req -days 3650 -in server.csr -signkey server.key -out
	// server.crt -extensions v3_req -extfile extfile.conf
	// where extfile.conf has the following details:
	// $ cat extfile.conf
	// [ v3_req ]
	// subjectAltName = IP:127.0.0.1
	// subjectKeyIdentifier=hash
	// authorityKeyIdentifier=keyid
	// basicConstraints = critical,CA:TRUE
	PublicKey2 = []byte(`-----BEGIN CERTIFICATE-----
MIICrDCCAjOgAwIBAgIUbFS1ugcEYYGQoTiV6O//r3wdO58wCgYIKoZIzj0EAwIw
gYQxCzAJBgNVBAYTAlVTMQswCQYDVQQIDAJOQzEQMA4GA1UEBwwHUmFsZWlnaDEQ
MA4GA1UECgwHUmVkIEhhdDEUMBIGA1UECwwLRW5naW5lZXJpbmcxDzANBgNVBAMM
BkNvcmVPUzEdMBsGCSqGSIb3DQEJARYOb2VtQGNvcmVvcy5jb20wHhcNMjAwNTA3
MjIzMzA3WhcNMzAwNTA1MjIzMzA3WjCBhDELMAkGA1UEBhMCVVMxCzAJBgNVBAgM
Ak5DMRAwDgYDVQQHDAdSYWxlaWdoMRAwDgYDVQQKDAdSZWQgSGF0MRQwEgYDVQQL
DAtFbmdpbmVlcmluZzEPMA0GA1UEAwwGQ29yZU9TMR0wGwYJKoZIhvcNAQkBFg5v
ZW1AY29yZW9zLmNvbTB2MBAGByqGSM49AgEGBSuBBAAiA2IABD8nQPQLeHZrnklT
f8QA17V3d7b+Z9yhcao/yyiIV1ie1EgLj5Xh2dEm4dFH4UdH/fnfUQh/ZZeyggmv
8hGFH+PS0B5eln74b4S9qihd/uBKGeFFgozi+I/Hj4OFI+OarKNkMGIwDwYDVR0R
BAgwBocEfwAAATAdBgNVHQ4EFgQUovVgWNFFPhrF7XzaRteDnpfPXxowHwYDVR0j
BBgwFoAUovVgWNFFPhrF7XzaRteDnpfPXxowDwYDVR0TAQH/BAUwAwEB/zAKBggq
hkjOPQQDAgNnADBkAjBvCIr9k43oR18Z4HLTzaRfzacFzo75Lt5n0pk3PA5CrUg3
sXU6o4IxyLNFHzIJn7cCMGTMVKEzoSZDclxkEgu53WM7PQljHgL9FJScEt4hzO2u
FFNjhq0ODV1LNc1i8pQCAg==
-----END CERTIFICATE-----`)
	// CABundle is a combination of PublicKey + PublicKey2.
	CABundle = append(append(PublicKey, '\n'), PublicKey2...)
)
