module github.com/coreos/ignition

go 1.13

require (
	github.com/ajeddeloh/go-json v0.0.0-20170920214419-6a2fe990e083
	github.com/aws/aws-sdk-go v1.19.11
	github.com/coreos/go-semver v0.3.0
	github.com/coreos/go-systemd v0.0.0-20181031085051-9002847aa142
	github.com/coreos/ign-converter v0.0.0-20200228175238-237c8512310a
	github.com/coreos/ignition/v2 v2.3.0
	github.com/godbus/dbus v4.1.0+incompatible // indirect
	github.com/pborman/uuid v0.0.0-20170612153648-e790cca94e6c
	github.com/pin/tftp v2.1.0+incompatible
	github.com/sigma/bdoor v0.0.0-20160202064022-babf2a4017b0 // indirect
	github.com/sigma/vmw-guestinfo v0.0.0-20160204083807-95dd4126d6e8
	github.com/smartystreets/goconvey v1.6.4 // indirect
	github.com/stretchr/testify v1.5.1
	github.com/vincent-petithory/dataurl v0.0.0-20160330182126-9a301d65acbb
	github.com/vmware/vmw-ovflib v0.0.0-20170608004843-1f217b9dc714
	go4.org v0.0.0-20200104003542-c7e774b10ea0
	golang.org/x/net v0.0.0-20190320064053-1272bf9dcd53
	golang.org/x/text v0.3.1-0.20190321115727-fe223c5a2583 // indirect
	gopkg.in/yaml.v2 v2.3.0 // indirect
)

replace github.com/coreos/ign-converter => github.com/LorbusChris/ign-converter v0.0.0-20200515140943-858ae6f84bec
