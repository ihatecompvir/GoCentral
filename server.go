package main

import (
	"github.com/ihatecompvir/nex-go"
)

func main() {
	nexServer := nex.NewServer()

	nexServer.SetPrudpVersion(0)
	nexServer.SetSignatureVersion(1)
	nexServer.SetKerberosKeySize(16)
	nexServer.SetChecksumVersion(1)
	nexServer.UsePacketCompression(true)
	nexServer.SetFlagsVersion(0)
	nexServer.SetAccessKey("bfa620c57c2d3bcdf4362a6fa6418e58")

	nexServer.Listen("0.0.0.0:16015")
}
