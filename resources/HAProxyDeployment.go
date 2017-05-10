package resources

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

const namespace string = "aslan"

type deploymentOptions struct {
	image string
	name  string
	port  int
}

func (options *deploymentOptions) createHAProxyDeployment(c *kubernetes.Clientset) error {
	appName := options.name
	// Define Deployments spec.
	deploySpec := &v1beta1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "extensions/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: appName,
		},
		Spec: v1beta1.DeploymentSpec{
			Replicas: int32p(1),
			Strategy: v1beta1.DeploymentStrategy{
				Type: v1beta1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &v1beta1.RollingUpdateDeployment{
					MaxUnavailable: &intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: int32(0),
					},
					MaxSurge: &intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: int32(1),
					},
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   appName,
					Labels: map[string]string{"app": appName},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						v1.Container{
							Name:  options.name,
							Image: options.image,
							Ports: []v1.ContainerPort{
								v1.ContainerPort{ContainerPort: int32(options.port), Protocol: v1.ProtocolTCP},
							},
							Args: []string{"--confd"},
							Env: []v1.EnvVar{
								v1.EnvVar{Name: "PATRONI_ETCD_HOST", Value: "http://etcd-cluster." + namespace + ":2379"},
							},
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									v1.ResourceCPU:    resource.MustParse("200m"),
									v1.ResourceMemory: resource.MustParse("512Mi"),
								},
							},
							ImagePullPolicy: v1.PullIfNotPresent,
						},
					},
					RestartPolicy: v1.RestartPolicyAlways,
					DNSPolicy:     v1.DNSClusterFirst,
				},
			},
		},
	}

	// Implement deployment update-or-create semantics.
	deploy := c.Extensions().Deployments(namespace)
	_, err := deploy.Create(deploySpec)
	switch {
	case err == nil:
		fmt.Println(options.name, " deployment is done in namespace ", namespace)
	case !errors.IsNotFound(err):
		return fmt.Errorf("could not create deployment: %s", err)
	default:
		_, err = deploy.Create(deploySpec)
		if err != nil {
			return fmt.Errorf("could not create deployment : %s", err)
		}
		fmt.Println("deployment is created")
	}

	return nil
}

func RunHaproxyDeployment(c *kubernetes.Clientset) {
	haProxyDepOps := deploymentOptions{
		image: "repo.emcrubicon.com/olifant-haproxy:1.0.0",
		name:  "haproxy-test",
		port:  5000,
	}
	haProxyDepOps.createHAProxyDeployment(c)
}

func int32p(i int32) *int32 {
	return &i
}
