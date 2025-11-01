package main

import (
	"context"
	"fmt"
	"log"
	"os"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeconfig string = os.Getenv("KUBECONFIG")
	namespace string = os.Getenv("NAMESPACE")
	appName string = os.Getenv("APP_NAME")
	domain string = os.Getenv("KYMA_DOMAIN")
	replicas int32 = 1
	action string = os.Getenv("ACTION")
)

func main() {
	// Load Kubernetes configuration
	config, err := getKubeConfig()
	if err != nil {
		log.Fatalf("Failed to get kubeconfig: %v", err)
	}

	// Create Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create clientset: %v", err)
	}

	// Create dynamic client for custom resources (APIRule)
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create dynamic client: %v", err)
	}

	ctx := context.Background()

	if action == "deploy" {
		// Step 1: Deploy Nginx
		fmt.Println("Creating Nginx Deployment...")
		if err := deployNginx(ctx, clientset, namespace, appName); err != nil {
			log.Fatalf("Failed to deploy Nginx: %v", err)
		}

		// Step 2: Create Service
		fmt.Println("Creating Service...")
		if err := createService(ctx, clientset, namespace, appName); err != nil {
			log.Fatalf("Failed to create service: %v", err)
		}

		// Step 3: Create Kyma APIRule
		fmt.Println("Creating Kyma APIRule...")
		if err := createAPIRule(ctx, dynamicClient, namespace, appName, domain); err != nil {
			log.Fatalf("Failed to create APIRule: %v", err)
		}

		fmt.Printf("\n✓ Successfully deployed Nginx server '%s' in namespace '%s'\n", appName, namespace)
		fmt.Printf("✓ Access URL: https://%s.%s\n", appName, domain)
		return
	}

	if action == "delete" {
		if err := deleteResources(ctx, clientset, dynamicClient, namespace, appName); err != nil {
			log.Fatalf("Failed to delete Resources: %v", err)
		}
		return
	}

	log.Fatal("choose an action delete or deploy!")
}

// getKubeConfig returns the Kubernetes configuration
func getKubeConfig() (*rest.Config, error) {
	// Try in-cluster config first (for running inside Kyma)
	config, err := rest.InClusterConfig()
	if err == nil {
		fmt.Println("Using in-cluster configuration")
		return config, nil
	}

	// Fall back to kubeconfig file
	if kubeconfig == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		kubeconfig = home + "/.kube/config"
	}

	fmt.Printf("Using kubeconfig from: %s\n", kubeconfig)
	config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build config: %w", err)
	}

	return config, nil
}

// deployNginx creates an Nginx deployment
func deployNginx(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string) error {

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app":     name,
				"version": "v1",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":     name,
						"version": "v1",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:1.25-alpine",
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 80,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("128Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("200m"),
									corev1.ResourceMemory: resource.MustParse("256Mi"),
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/",
										Port: intstr.FromInt(80),
									},
								},
								InitialDelaySeconds: 10,
								PeriodSeconds:       10,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/",
										Port: intstr.FromInt(80),
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       5,
							},
						},
					},
				},
			},
		},
	}

	deploymentsClient := clientset.AppsV1().Deployments(namespace)

	// Check if deployment exists
	_, err := deploymentsClient.Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		// Update existing deployment
		result, err := deploymentsClient.Update(ctx, deployment, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update deployment: %w", err)
		}
		fmt.Printf("  ✓ Updated deployment: %s\n", result.GetObjectMeta().GetName())
	} else {
		// Create new deployment
		result, err := deploymentsClient.Create(ctx, deployment, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create deployment: %w", err)
		}
		fmt.Printf("  ✓ Created deployment: %s\n", result.GetObjectMeta().GetName())
	}

	return nil
}

// deleteResources deletes the Deployment, Service, and APIRule
func deleteResources(ctx context.Context, clientset *kubernetes.Clientset, dynamicClient dynamic.Interface, namespace, name string) error {
	fmt.Println("Deleting resources...")

	// Delete Deployment
	fmt.Println("Deleting Deployment...")
	deploymentsClient := clientset.AppsV1().Deployments(namespace)
	deletePolicy := metav1.DeletePropagationForeground
	err := deploymentsClient.Delete(ctx, name, metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	})
	if err != nil {
		fmt.Printf("  ⚠ Warning: Failed to delete deployment: %v\n", err)
	} else {
		fmt.Printf("  ✓ Deleted deployment: %s\n", name)
	}

	// Delete Service
	fmt.Println("Deleting Service...")
	servicesClient := clientset.CoreV1().Services(namespace)
	err = servicesClient.Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		fmt.Printf("  ⚠ Warning: Failed to delete service: %v\n", err)
	} else {
		fmt.Printf("  ✓ Deleted service: %s\n", name)
	}

	// Delete APIRule
	fmt.Println("Deleting APIRule...")
	apiRuleGVR := schema.GroupVersionResource{
		Group:    "gateway.kyma-project.io",
		Version:  "v2alpha1",
		Resource: "apirules",
	}
	apiRuleClient := dynamicClient.Resource(apiRuleGVR).Namespace(namespace)
	err = apiRuleClient.Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		fmt.Printf("  ⚠ Warning: Failed to delete APIRule: %v\n", err)
	} else {
		fmt.Printf("  ✓ Deleted APIRule: %s\n", name)
	}

	return nil
}

// createService creates a Service for Nginx
func createService(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": name,
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       80,
					TargetPort: intstr.FromInt(80),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: map[string]string{
				"app": name,
			},
		},
	}

	servicesClient := clientset.CoreV1().Services(namespace)

	// Check if service exists
	_, err := servicesClient.Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		// Update existing service
		result, err := servicesClient.Update(ctx, service, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update service: %w", err)
		}
		fmt.Printf("  ✓ Updated service: %s\n", result.GetObjectMeta().GetName())
	} else {
		// Create new service
		result, err := servicesClient.Create(ctx, service, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create service: %w", err)
		}
		fmt.Printf("  ✓ Created service: %s\n", result.GetObjectMeta().GetName())
	}

	return nil
}

// createAPIRule creates a Kyma APIRule for external access
func createAPIRule(ctx context.Context, dynamicClient dynamic.Interface, namespace, name, domain string) error {
	// Define the APIRule GVR (Group, Version, Resource)
	apiRuleGVR := schema.GroupVersionResource{
		Group:    "gateway.kyma-project.io",
		Version:  "v2alpha1",
		Resource: "apirules",
	}

	// Create the APIRule object (v2alpha1 format)
	apiRule := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "gateway.kyma-project.io/v2alpha1",
			"kind":       "APIRule",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
				"labels": map[string]interface{}{
					"app": name,
				},
			},
			"spec": map[string]interface{}{
				"gateway": "kyma-system/kyma-gateway",
				"hosts": []interface{}{
					fmt.Sprintf("%s.%s", name, domain),
				},
				"service": map[string]interface{}{
					"name": name,
					"port": 80,
				},
				"rules": []interface{}{
					map[string]interface{}{
						"path": "/*",
						"methods": []interface{}{
							"GET",
							"POST",
							"PUT",
							"DELETE",
							"PATCH",
							"HEAD",
						},
						"noAuth": true,
					},
				},
			},
		},
	}

	apiRuleClient := dynamicClient.Resource(apiRuleGVR).Namespace(namespace)

	// Check if APIRule exists
	existing, err := apiRuleClient.Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		// Update existing APIRule - preserve resourceVersion
		apiRule.SetResourceVersion(existing.GetResourceVersion())
		result, err := apiRuleClient.Update(ctx, apiRule, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update APIRule: %w", err)
		}
		fmt.Printf("  ✓ Updated APIRule: %s\n", result.GetName())
	} else {
		// Create new APIRule
		result, err := apiRuleClient.Create(ctx, apiRule, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create APIRule: %w", err)
		}
		fmt.Printf("  ✓ Created APIRule: %s\n", result.GetName())
	}

	return nil
}