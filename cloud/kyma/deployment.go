package kyma

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// deployDeployment creates or updates a Kubernetes Deployment for the application.
func (c *client) deployDeployment(ctx context.Context, app *Application) error {
	deployment := c.buildDeployment(app)

	deploymentsClient := c.clientset.AppsV1().Deployments(app.Namespace)

	// Check if deployment exists
	existing, err := deploymentsClient.Get(ctx, app.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Create new deployment
			_, err := deploymentsClient.Create(ctx, deployment, metav1.CreateOptions{})
			if err != nil {
				return NewResourceError("create", "Deployment", app.Name, app.Namespace, err)
			}
			return nil
		}
		return NewResourceError("get", "Deployment", app.Name, app.Namespace, err)
	}

	// Update existing deployment - preserve resource version
	deployment.ResourceVersion = existing.ResourceVersion
	_, err = deploymentsClient.Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return NewResourceError("update", "Deployment", app.Name, app.Namespace, err)
	}

	return nil
}

// deleteDeployment removes a Kubernetes Deployment.
func (c *client) deleteDeployment(ctx context.Context, namespace, name string) error {
	deploymentsClient := c.clientset.AppsV1().Deployments(namespace)

	deletePolicy := metav1.DeletePropagationForeground
	err := deploymentsClient.Delete(ctx, name, metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	})

	if err != nil {
		if errors.IsNotFound(err) {
			// Already deleted, not an error
			return nil
		}
		return NewResourceError("delete", "Deployment", name, namespace, err)
	}

	return nil
}

// getDeploymentStatus retrieves the status of a Deployment and updates the ApplicationStatus.
func (c *client) getDeploymentStatus(ctx context.Context, namespace, name string, status *ApplicationStatus) error {
	deploymentsClient := c.clientset.AppsV1().Deployments(namespace)

	deployment, err := deploymentsClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			status.DeploymentReady = false
			return nil
		}
		return err
	}

	status.DesiredReplicas = *deployment.Spec.Replicas
	status.ReadyReplicas = deployment.Status.ReadyReplicas
	status.DeploymentReady = deployment.Status.ReadyReplicas == *deployment.Spec.Replicas

	// Add conditions
	for _, condition := range deployment.Status.Conditions {
		status.Conditions = append(status.Conditions, StatusCondition{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			Reason:             condition.Reason,
			Message:            condition.Message,
			LastTransitionTime: condition.LastTransitionTime.Time,
		})
	}

	return nil
}

// buildDeployment constructs a Kubernetes Deployment resource from Application config.
func (c *client) buildDeployment(app *Application) *appsv1.Deployment {
	labels := make(map[string]string)
	for k, v := range app.Labels {
		labels[k] = v
	}
	labels["app"] = app.Name

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        app.Name,
			Namespace:   app.Namespace,
			Labels:      labels,
			Annotations: app.Annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &app.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": app.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: app.Annotations,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						c.buildContainer(app),
					},
				},
			},
		},
	}

	return deployment
}

// buildContainer constructs a Container specification from Application config.
func (c *client) buildContainer(app *Application) corev1.Container {
	container := corev1.Container{
		Name:  app.Name,
		Image: app.Image,
		Ports: []corev1.ContainerPort{
			{
				Name:          "http",
				ContainerPort: app.ContainerPort,
				Protocol:      corev1.ProtocolTCP,
			},
		},
	}

	// Add resource requirements if specified
	if app.Resources.RequestsCPU != "" || app.Resources.RequestsMemory != "" ||
		app.Resources.LimitsCPU != "" || app.Resources.LimitsMemory != "" {
		container.Resources = c.buildResourceRequirements(app.Resources)
	}

	// Add liveness probe if specified
	if app.LivenessProbe != nil {
		container.LivenessProbe = c.buildProbe(app.LivenessProbe)
	}

	// Add readiness probe if specified
	if app.ReadinessProbe != nil {
		container.ReadinessProbe = c.buildProbe(app.ReadinessProbe)
	}

	// Add environment variables if specified
	if len(app.Env) > 0 {
		container.Env = c.buildEnvVars(app.Env)
	}

	return container
}

// buildResourceRequirements converts ResourceRequirements to Kubernetes format.
func (c *client) buildResourceRequirements(reqs ResourceRequirements) corev1.ResourceRequirements {
	resources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{},
		Limits:   corev1.ResourceList{},
	}

	if reqs.RequestsCPU != "" {
		resources.Requests[corev1.ResourceCPU] = resource.MustParse(reqs.RequestsCPU)
	}
	if reqs.RequestsMemory != "" {
		resources.Requests[corev1.ResourceMemory] = resource.MustParse(reqs.RequestsMemory)
	}
	if reqs.LimitsCPU != "" {
		resources.Limits[corev1.ResourceCPU] = resource.MustParse(reqs.LimitsCPU)
	}
	if reqs.LimitsMemory != "" {
		resources.Limits[corev1.ResourceMemory] = resource.MustParse(reqs.LimitsMemory)
	}

	return resources
}

// buildProbe converts HealthProbe to Kubernetes Probe format.
func (c *client) buildProbe(probe *HealthProbe) *corev1.Probe {
	k8sProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: probe.Path,
				Port: intstr.FromInt32(probe.Port),
			},
		},
	}

	if probe.InitialDelaySeconds > 0 {
		k8sProbe.InitialDelaySeconds = probe.InitialDelaySeconds
	}
	if probe.PeriodSeconds > 0 {
		k8sProbe.PeriodSeconds = probe.PeriodSeconds
	}
	if probe.TimeoutSeconds > 0 {
		k8sProbe.TimeoutSeconds = probe.TimeoutSeconds
	}
	if probe.SuccessThreshold > 0 {
		k8sProbe.SuccessThreshold = probe.SuccessThreshold
	}
	if probe.FailureThreshold > 0 {
		k8sProbe.FailureThreshold = probe.FailureThreshold
	}

	return k8sProbe
}

// buildEnvVars converts environment variables map to Kubernetes format.
func (c *client) buildEnvVars(env map[string]string) []corev1.EnvVar {
	envVars := make([]corev1.EnvVar, 0, len(env))
	for key, value := range env {
		envVars = append(envVars, corev1.EnvVar{
			Name:  key,
			Value: value,
		})
	}
	return envVars
}
