package main

import (
	"context"
	"fmt"
	"log"
	"os"

	clinicdomain "example.com/clinic-domain-onboard"
)

func main() {
	if len(os.Args) != 5 {
		log.Fatal("usage: domain-onboard DOMAIN TXT_VALUE WEBHOOK_URL WEBHOOK_SECRET")
	}
	client, err := clinicdomain.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	result, err := client.OnboardClinicDomain(context.Background(), clinicdomain.OnboardingInput{
		Domain: os.Args[1], Verification: os.Args[2], WebhookURL: os.Args[3], WebhookSecret: os.Args[4],
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("zone_id=%s state=%s\n", result.ZoneID, result.State)
}
