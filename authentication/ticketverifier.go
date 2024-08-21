package authentication

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/url"
)

type TicketVerifier struct {
	TicketVerifierEndpoint string
}

func (verifier *TicketVerifier) VerifyTicket(ticketData []byte, consoleType int) bool {
	encodedTicket := base64.StdEncoding.EncodeToString(ticketData)
	urlEncodedTicket := url.QueryEscape(encodedTicket)

	verifyURL := fmt.Sprintf("%s?ticket=%s&platform=%d", verifier.TicketVerifierEndpoint, urlEncodedTicket, consoleType)
	resp, err := http.Get(verifyURL)
	if err != nil {
		log.Println("Failed to verify ticket:", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true
	} else {
		return false
	}
}
