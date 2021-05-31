package adalwrapper_test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/arm/resources/2020-06-01/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/armcore"
	"github.com/Azure/azure-sdk-for-go/sdk/to"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/magodo/adalwrapper"
)

func wrapServicePrincipalToken(oauthConfig adal.OAuthConfig, clientId, clientSecret, resource string, sender autorest.Sender) (*adalwrapper.TokenCredential, error) {
	spt, err := adal.NewServicePrincipalToken(oauthConfig, clientId, clientSecret, resource)
	if err != nil {
		return nil, err
	}
	spt.SetSender(sender)
	return adalwrapper.NewTokenCredential(autorest.NewBearerAuthorizer(spt)), nil
}

func ExampleCreateResourceGroup() {
	subscriptionId, ok := os.LookupEnv("ARM_SUBSCRIPTION_ID")
	if !ok {
		log.Fatal(`"ARM_SUBSCRIPTION_ID" is not set`)
	}
	tenantId, ok := os.LookupEnv("ARM_TENANT_ID")
	if !ok {
		log.Fatal(`"ARM_TENANT_ID" is not set`)
	}
	clientId, ok := os.LookupEnv("ARM_CLIENT_ID")
	if !ok {
		log.Fatal(`"ARM_CLIENT_ID" is not set`)
	}
	clientSecret, ok := os.LookupEnv("ARM_CLIENT_SECRET")
	if !ok {
		log.Fatal(`"ARM_CLIENT_SECRET" is not set`)
	}

	// Setup the sender.
	// Same as 'sender.BuildSender("AzureRM")', except without logging.
	sender := autorest.DecorateSender(&http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	})

	// Setup the oauth config
	oauthConfig, err := adal.NewOAuthConfig("https://login.microsoftonline.com/", tenantId)
	if err != nil {
		log.Fatal(err)
	}

	resource := "https://management.azure.com/"

	cred, err := wrapServicePrincipalToken(*oauthConfig, clientId, clientSecret, resource, sender)
	if err != nil {
		log.Fatal(err)
	}

	// Use of the wrapped token credential to create and then destroy a resource group
	client := armresources.NewResourceGroupsClient(
		armcore.NewDefaultConnection(
			cred,
			&armcore.ConnectionOptions{
				DisableRPRegistration: true,
			}),
		subscriptionId)

	const rgName = "adalwrapper-rg"
	if _, err = client.CreateOrUpdate(context.Background(), rgName, armresources.ResourceGroup{
		Location: to.StringPtr("eastus2"),
	}, nil); err != nil {
		log.Fatalf("failed to create the resource group: %v", err)
	}

	poller, err := client.BeginDelete(context.Background(), rgName, nil)
	if err != nil {
		log.Fatalf("failed to start to destroy the resource group: %v", err)
	}

	if _, err := poller.PollUntilDone(context.Background(), 5*time.Second); err != nil {
		log.Fatalf("failed to destroy the resource group: %v", err)
	}
	fmt.Println("OK")
	// Output: OK
}
