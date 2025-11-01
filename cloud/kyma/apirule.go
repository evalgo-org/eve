package kyma

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// APIRule GVR (Group, Version, Resource) for Kyma APIRule custom resource
var apiRuleGVR = schema.GroupVersionResource{
	Group:    "gateway.kyma-project.io",
	Version:  "v2alpha1",
	Resource: "apirules",
}

// deployAPIRule creates or updates a Kyma APIRule for external access.
func (c *client) deployAPIRule(ctx context.Context, app *Application) error {
	if app.Domain == "" {
		return NewValidationError("Domain", "domain is required for APIRule creation")
	}

	apiRule := c.buildAPIRule(app)

	apiRuleClient := c.dynamicClient.Resource(apiRuleGVR).Namespace(app.Namespace)

	// Check if APIRule exists
	existing, err := apiRuleClient.Get(ctx, app.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Create new APIRule
			_, err := apiRuleClient.Create(ctx, apiRule, metav1.CreateOptions{})
			if err != nil {
				return NewResourceError("create", "APIRule", app.Name, app.Namespace, err)
			}
			return nil
		}
		return NewResourceError("get", "APIRule", app.Name, app.Namespace, err)
	}

	// Update existing APIRule - preserve resource version
	apiRule.SetResourceVersion(existing.GetResourceVersion())
	_, err = apiRuleClient.Update(ctx, apiRule, metav1.UpdateOptions{})
	if err != nil {
		return NewResourceError("update", "APIRule", app.Name, app.Namespace, err)
	}

	return nil
}

// deleteAPIRule removes a Kyma APIRule.
func (c *client) deleteAPIRule(ctx context.Context, namespace, name string) error {
	apiRuleClient := c.dynamicClient.Resource(apiRuleGVR).Namespace(namespace)

	err := apiRuleClient.Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Already deleted, not an error
			return nil
		}
		return NewResourceError("delete", "APIRule", name, namespace, err)
	}

	return nil
}

// getAPIRuleStatus retrieves the status of an APIRule and updates the ApplicationStatus.
func (c *client) getAPIRuleStatus(ctx context.Context, namespace, name string, status *ApplicationStatus) error {
	apiRuleClient := c.dynamicClient.Resource(apiRuleGVR).Namespace(namespace)

	apiRule, err := apiRuleClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			status.APIRuleExists = false
			status.APIRuleReady = false
			return nil
		}
		return err
	}

	status.APIRuleExists = true

	// Extract status from APIRule
	statusObj, found, err := unstructured.NestedMap(apiRule.Object, "status")
	if err != nil || !found {
		status.APIRuleReady = false
		return nil
	}

	// Check if APIRule is ready
	state, found, err := unstructured.NestedString(statusObj, "state")
	if err != nil || !found {
		status.APIRuleReady = false
		return nil
	}

	status.APIRuleReady = (state == "Ready")

	// Extract URL from spec
	hosts, found, err := unstructured.NestedStringSlice(apiRule.Object, "spec", "hosts")
	if err == nil && found && len(hosts) > 0 {
		status.URL = fmt.Sprintf("https://%s", hosts[0])
	}

	return nil
}

// buildAPIRule constructs a Kyma APIRule custom resource from Application config.
func (c *client) buildAPIRule(app *Application) *unstructured.Unstructured {
	labels := make(map[string]interface{})
	for k, v := range app.Labels {
		labels[k] = v
	}
	labels["app"] = app.Name

	// Convert methods to interface slice
	methods := make([]interface{}, len(app.Methods))
	for i, method := range app.Methods {
		methods[i] = method
	}

	// Build the rules array
	rules := []interface{}{
		map[string]interface{}{
			"path":    app.PathPrefix,
			"methods": methods,
			"noAuth":  !app.AuthEnabled,
		},
	}

	// Build the APIRule object
	apiRule := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "gateway.kyma-project.io/v2alpha1",
			"kind":       "APIRule",
			"metadata": map[string]interface{}{
				"name":      app.Name,
				"namespace": app.Namespace,
				"labels":    labels,
			},
			"spec": map[string]interface{}{
				"gateway": "kyma-system/kyma-gateway",
				"hosts": []interface{}{
					fmt.Sprintf("%s.%s", app.Name, app.Domain),
				},
				"service": map[string]interface{}{
					"name": app.Name,
					"port": app.ServicePort,
				},
				"rules": rules,
			},
		},
	}

	// Add annotations if present
	if len(app.Annotations) > 0 {
		annotations := make(map[string]interface{})
		for k, v := range app.Annotations {
			annotations[k] = v
		}
		metadata := apiRule.Object["metadata"].(map[string]interface{})
		metadata["annotations"] = annotations
	}

	return apiRule
}
