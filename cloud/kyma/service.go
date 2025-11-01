package kyma

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// deployService creates or updates a Kubernetes Service for the application.
func (c *client) deployService(ctx context.Context, app *Application) error {
	service := c.buildService(app)

	servicesClient := c.clientset.CoreV1().Services(app.Namespace)

	// Check if service exists
	existing, err := servicesClient.Get(ctx, app.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Create new service
			_, err := servicesClient.Create(ctx, service, metav1.CreateOptions{})
			if err != nil {
				return NewResourceError("create", "Service", app.Name, app.Namespace, err)
			}
			return nil
		}
		return NewResourceError("get", "Service", app.Name, app.Namespace, err)
	}

	// Update existing service - preserve resource version and cluster IP
	service.ResourceVersion = existing.ResourceVersion
	service.Spec.ClusterIP = existing.Spec.ClusterIP
	_, err = servicesClient.Update(ctx, service, metav1.UpdateOptions{})
	if err != nil {
		return NewResourceError("update", "Service", app.Name, app.Namespace, err)
	}

	return nil
}

// deleteService removes a Kubernetes Service.
func (c *client) deleteService(ctx context.Context, namespace, name string) error {
	servicesClient := c.clientset.CoreV1().Services(namespace)

	err := servicesClient.Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Already deleted, not an error
			return nil
		}
		return NewResourceError("delete", "Service", name, namespace, err)
	}

	return nil
}

// getServiceStatus retrieves the status of a Service and updates the ApplicationStatus.
func (c *client) getServiceStatus(ctx context.Context, namespace, name string, status *ApplicationStatus) error {
	servicesClient := c.clientset.CoreV1().Services(namespace)

	_, err := servicesClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			status.ServiceExists = false
			return nil
		}
		return err
	}

	status.ServiceExists = true
	return nil
}

// buildService constructs a Kubernetes Service resource from Application config.
func (c *client) buildService(app *Application) *corev1.Service {
	labels := make(map[string]string)
	for k, v := range app.Labels {
		labels[k] = v
	}
	labels["app"] = app.Name

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        app.Name,
			Namespace:   app.Namespace,
			Labels:      labels,
			Annotations: app.Annotations,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       app.ServicePort,
					TargetPort: intstr.FromInt32(app.ContainerPort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: map[string]string{
				"app": app.Name,
			},
		},
	}

	return service
}
