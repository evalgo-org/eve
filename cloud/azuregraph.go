package cloud

import (
	"context"
	azidentity "github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	msgraphcore "github.com/microsoftgraph/msgraph-sdk-go-core"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/users"

	eve "eve.evalgo.org/common"
)

func ptrInt32(i int32) *int32 {
	return &i
}

func AzureEmails(tenantId string, clientId string, clientSecret string) error {
	cred, err := azidentity.NewClientSecretCredential(
		tenantId,
		clientId,
		clientSecret,
		nil,
	)
	if err != nil {
		eve.Logger.Info("Error creating credentials: ", err)
		return err
	}
	graphClient, _ := msgraphsdk.NewGraphServiceClientWithCredentials(cred, []string{"https://graph.microsoft.com/.default"})
	opts := &users.ItemMailFoldersItemMessagesRequestBuilderGetRequestConfiguration{
		QueryParameters: &users.ItemMailFoldersItemMessagesRequestBuilderGetQueryParameters{
			Top:    ptrInt32(10), // Limit to 10 messages
			Select: []string{"subject", "receivedDateTime"},
		},
	}
	resp, err := graphClient.Users().
		ByUserId("francisc@simon.services").
		MailFolders().
		ByMailFolderId("inbox").
		Messages().
		Get(context.Background(), opts)
	if err != nil {
		return err
	}
	for _, msg := range resp.GetValue() {
		eve.Logger.Info("Subject:", *msg.GetSubject())
		eve.Logger.Info("Received:", *msg.GetReceivedDateTime())
		eve.Logger.Info("---")
	}
	return nil
}

func AzureCalendar(tenantId string, clientId string, clientSecret string, email string, start string, end string) error {
	cred, err := azidentity.NewClientSecretCredential(
		tenantId,
		clientId,
		clientSecret,
		nil,
	)
	if err != nil {
		eve.Logger.Info("Error creating credentials:", err)
		return err
	}
	graphClient, _ := msgraphsdk.NewGraphServiceClientWithCredentials(cred, []string{"https://graph.microsoft.com/.default"})
	query := &users.ItemCalendarViewRequestBuilderGetQueryParameters{
		StartDateTime: &start,
		EndDateTime:   &end,
		Top:           ptrInt32(10),
		Select:        []string{"subject", "start", "end"},
	}
	opts := &users.ItemCalendarViewRequestBuilderGetRequestConfiguration{
		QueryParameters: query,
	}
	eventsResponse, err := graphClient.Users().
		ByUserId(email).
		CalendarView().
		Get(context.Background(), opts)
	if err != nil {
		panic(err)
	}
	eit, err := msgraphcore.NewPageIterator[models.Eventable](
		eventsResponse,
		graphClient.GetAdapter(),
		models.CreateEventCollectionResponseFromDiscriminatorValue,
	)
	if err != nil {
		panic(err)
	}
	err = eit.Iterate(context.Background(), func(ev models.Eventable) bool {
		eve.Logger.Info(" TIME: ", *ev.GetStart().GetDateTime(), " => ", *ev.GetEnd().GetDateTime(), " Subject: ", *ev.GetSubject())
		return true
	})

	return nil
}
