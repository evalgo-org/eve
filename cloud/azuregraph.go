// Package cloud provides Microsoft Azure cloud service integrations for the EVE evaluation system.
// This package focuses on Microsoft Graph API operations for accessing Office 365 services
// including email (Exchange Online) and calendar (Outlook) functionality.
//
// The package implements secure, authenticated access to Microsoft Graph API using
// client credentials flow (application permissions) suitable for service-to-service
// scenarios and automated data processing workflows.
//
// Microsoft Graph Integration:
//   - Authentication via Azure Active Directory client credentials
//   - Email access through Exchange Online Graph endpoints
//   - Calendar access through Outlook Graph endpoints
//   - Pagination support for large result sets
//   - Structured data retrieval with configurable field selection
//
// Authentication Requirements:
//   - Valid Azure tenant with registered application
//   - Application must have appropriate Graph API permissions
//   - Client secret or certificate for secure authentication
//   - Admin consent for application permissions
//
// Required Graph API Permissions:
//
//	For Email Operations:
//	  - Mail.Read or Mail.ReadWrite (application permission)
//	  - User.Read.All (for accessing specific user mailboxes)
//
//	For Calendar Operations:
//	  - Calendars.Read or Calendars.ReadWrite (application permission)
//	  - User.Read.All (for accessing specific user calendars)
//
// Security Considerations:
//   - Client secrets should be stored securely (Azure Key Vault recommended)
//   - Use principle of least privilege for API permissions
//   - Implement proper audit logging for data access
//   - Consider certificate-based authentication for production
//   - Regular rotation of client secrets and certificates
//
// Rate Limiting:
//
//	Microsoft Graph API implements throttling limits that may affect
//	high-volume operations. The package should be used with appropriate
//	retry logic and respect for API rate limits.
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

// ptrInt32 creates a pointer to an int32 value for use with Microsoft Graph API.
// The Microsoft Graph SDK requires pointer values for optional parameters,
// and this utility function simplifies the creation of int32 pointers.
//
// This is a common pattern in Go APIs where optional parameters are
// represented as pointers, allowing nil to indicate "not specified".
//
// Parameters:
//   - i: int32 value to convert to pointer
//
// Returns:
//   - *int32: Pointer to the provided int32 value
//
// Usage:
//
//	Used internally for setting query parameters like Top (limit)
//	in Microsoft Graph API requests where the SDK expects pointer types.
func ptrInt32(i int32) *int32 {
	return &i
}

// AzureEmails retrieves email messages from a specified user's inbox using Microsoft Graph API.
// This function implements secure authentication and retrieves the most recent emails
// with configurable field selection for efficient data transfer.
//
// Authentication Flow:
//
//	Uses Azure AD client credentials flow for service-to-service authentication.
//	This requires a registered Azure application with appropriate Graph API permissions
//	and admin consent for accessing user mailboxes.
//
// Email Retrieval Features:
//   - Accesses user's inbox folder specifically
//   - Retrieves top 10 most recent messages (configurable)
//   - Returns selected fields only (subject, receivedDateTime) for efficiency
//   - Structured logging of email metadata
//
// Required Azure Permissions:
//   - Mail.Read (application permission) - to read user mailboxes
//   - User.Read.All (application permission) - to access specific users
//   - Admin consent must be granted for application permissions
//
// Data Privacy:
//
//	This function accesses sensitive email data. Ensure compliance with:
//	- GDPR and data protection regulations
//	- Company data handling policies
//	- Audit and logging requirements
//	- User consent and notification policies
//
// Parameters:
//   - tenantId: Azure Active Directory tenant ID (GUID format)
//   - clientId: Registered application client ID (GUID format)
//   - clientSecret: Application client secret for authentication
//
// Returns:
//   - error: nil on success, error details on authentication or API failures
//
// Error Conditions:
//   - Invalid tenant ID, client ID, or client secret
//   - Insufficient permissions for mailbox access
//   - Network connectivity issues to Microsoft Graph API
//   - User mailbox not found or inaccessible
//   - API rate limiting or throttling
//
// Output Format:
//
//	Logs email information using eve.Logger with:
//	- Subject: Email subject line
//	- Received: Email received timestamp
//	- Separator line for readability
//
// Security Notes:
//   - Client secrets should never be hardcoded
//   - Use Azure Key Vault or similar for secret management
//   - Implement proper audit logging for email access
//   - Consider using managed identities in Azure environments
//
// Example Usage:
//
//	err := AzureEmails("tenant-id", "client-id", "client-secret")
//	if err != nil {
//	    log.Printf("Failed to retrieve emails: %v", err)
//	}
//
// Performance Considerations:
//   - Limited to 10 messages for efficiency (hardcoded)
//   - Only retrieves essential fields to minimize data transfer
//   - Consider implementing pagination for larger result sets
//   - Cache authentication tokens to avoid repeated auth overhead
func AzureEmails(tenantId string, clientId string, clientSecret string) error {
	// Create Azure AD client credentials for service authentication
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

	// Initialize Microsoft Graph client with appropriate scopes
	graphClient, _ := msgraphsdk.NewGraphServiceClientWithCredentials(
		cred,
		[]string{"https://graph.microsoft.com/.default"},
	)

	// Configure request parameters for email retrieval
	opts := &users.ItemMailFoldersItemMessagesRequestBuilderGetRequestConfiguration{
		QueryParameters: &users.ItemMailFoldersItemMessagesRequestBuilderGetQueryParameters{
			Top:    ptrInt32(10),                            // Limit to 10 messages
			Select: []string{"subject", "receivedDateTime"}, // Select only required fields
		},
	}

	// Execute email retrieval request for specific user's inbox
	resp, err := graphClient.Users().
		ByUserId("francisc@simon.services"). // Target user email address
		MailFolders().
		ByMailFolderId("inbox"). // Access inbox folder specifically
		Messages().
		Get(context.Background(), opts)

	if err != nil {
		return err
	}

	// Process and log email information
	for _, msg := range resp.GetValue() {
		eve.Logger.Info("Subject:", *msg.GetSubject())
		eve.Logger.Info("Received:", *msg.GetReceivedDateTime())
		eve.Logger.Info("---")
	}

	return nil
}

// AzureCalendar retrieves calendar events for a specified user within a date range using Microsoft Graph API.
// This function provides comprehensive calendar access with date filtering, pagination support,
// and efficient field selection for calendar event retrieval.
//
// Calendar Access Features:
//   - Date range filtering with start and end parameters
//   - Pagination support for large calendar datasets
//   - Configurable result limits (currently 10 events)
//   - Selective field retrieval for performance optimization
//   - Iterator pattern for processing large result sets
//
// Authentication Flow:
//
//	Uses the same Azure AD client credentials flow as AzureEmails,
//	requiring appropriate calendar permissions and admin consent.
//
// Required Azure Permissions:
//   - Calendars.Read (application permission) - to read user calendars
//   - User.Read.All (application permission) - to access specific users
//   - Admin consent must be granted for application permissions
//
// Date Range Processing:
//
//	The function accepts start and end date parameters in ISO 8601 format
//	and retrieves calendar events within the specified time window.
//
// Pagination Implementation:
//
//	Uses Microsoft Graph SDK's PageIterator for efficient processing
//	of large calendar datasets without loading all data into memory.
//
// Parameters:
//   - tenantId: Azure Active Directory tenant ID (GUID format)
//   - clientId: Registered application client ID (GUID format)
//   - clientSecret: Application client secret for authentication
//   - email: Target user's email address for calendar access
//   - start: Start date for event retrieval (ISO 8601 format: "2024-01-01T00:00:00Z")
//   - end: End date for event retrieval (ISO 8601 format: "2024-01-31T23:59:59Z")
//
// Returns:
//   - error: nil on success, error details on authentication or API failures
//
// Error Conditions:
//   - Invalid authentication credentials
//   - Insufficient permissions for calendar access
//   - Invalid date format in start/end parameters
//   - User calendar not found or inaccessible
//   - Network connectivity issues to Microsoft Graph API
//   - API rate limiting or throttling
//
// Date Format Requirements:
//
//	Start and end parameters must be in ISO 8601 format with timezone:
//	- Valid: "2024-01-01T00:00:00Z"
//	- Valid: "2024-01-01T00:00:00.000Z"
//	- Invalid: "2024-01-01" (missing time component)
//	- Invalid: "01/01/2024" (wrong format)
//
// Output Format:
//
//	Logs calendar event information using eve.Logger with:
//	- TIME: Event start and end timestamps
//	- Subject: Event title/subject
//	- Structured format for parsing and analysis
//
// Performance Optimization:
//   - Limited result set (Top: 10) for initial retrieval
//   - Selective field retrieval (subject, start, end only)
//   - Iterator pattern prevents memory overload with large calendars
//   - Efficient pagination through Microsoft Graph SDK
//
// Security Considerations:
//   - Calendar data may contain sensitive meeting information
//   - Implement appropriate access controls and audit logging
//   - Consider data retention and privacy policies
//   - Use secure credential storage (Azure Key Vault recommended)
//
// Example Usage:
//
//	start := "2024-01-01T00:00:00Z"
//	end := "2024-01-31T23:59:59Z"
//	err := AzureCalendar("tenant-id", "client-id", "client-secret",
//	                    "user@company.com", start, end)
//	if err != nil {
//	    log.Printf("Failed to retrieve calendar: %v", err)
//	}
//
// Integration Scenarios:
//   - Meeting attendance tracking and reporting
//   - Resource utilization analysis
//   - Automated scheduling conflict detection
//   - Calendar synchronization with external systems
//   - Compliance and audit reporting for meeting data
//
// Scalability Notes:
//   - Consider implementing caching for frequently accessed calendar data
//   - Use background processing for large calendar synchronization
//   - Implement retry logic for transient API failures
//   - Monitor API usage to stay within rate limits
func AzureCalendar(tenantId string, clientId string, clientSecret string, email string, start string, end string) error {
	// Create Azure AD client credentials for service authentication
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

	// Initialize Microsoft Graph client with appropriate scopes
	graphClient, _ := msgraphsdk.NewGraphServiceClientWithCredentials(
		cred,
		[]string{"https://graph.microsoft.com/.default"},
	)

	// Configure query parameters for calendar event retrieval
	query := &users.ItemCalendarViewRequestBuilderGetQueryParameters{
		StartDateTime: &start,                              // Filter by start date
		EndDateTime:   &end,                                // Filter by end date
		Top:           ptrInt32(10),                        // Limit initial result set
		Select:        []string{"subject", "start", "end"}, // Select only required fields
	}

	// Configure request options with query parameters
	opts := &users.ItemCalendarViewRequestBuilderGetRequestConfiguration{
		QueryParameters: query,
	}

	// Execute calendar view request for specified user and date range
	eventsResponse, err := graphClient.Users().
		ByUserId(email). // Target user specified by email
		CalendarView().  // Use calendar view for date range filtering
		Get(context.Background(), opts)

	if err != nil {
		panic(err) // Note: Consider replacing panic with proper error handling
	}

	// Create page iterator for efficient processing of large result sets
	eit, err := msgraphcore.NewPageIterator[models.Eventable](
		eventsResponse,
		graphClient.GetAdapter(),
		models.CreateEventCollectionResponseFromDiscriminatorValue,
	)
	if err != nil {
		panic(err) // Note: Consider replacing panic with proper error handling
	}

	// Iterate through all calendar events with pagination support
	_ = eit.Iterate(context.Background(), func(ev models.Eventable) bool {
		// Log event information with structured format
		eve.Logger.Info(" TIME: ", *ev.GetStart().GetDateTime(), " => ",
			*ev.GetEnd().GetDateTime(), " Subject: ", *ev.GetSubject())
		return true // Continue iteration (return false to stop)
	})

	return nil
}
