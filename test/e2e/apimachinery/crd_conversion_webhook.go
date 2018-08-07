/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package apimachinery

import (
	"time"

	apps "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	rbacv1beta1 "k8s.io/api/rbac/v1beta1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"
	utilversion "k8s.io/kubernetes/pkg/util/version"
	"k8s.io/kubernetes/test/e2e/framework"
	imageutils "k8s.io/kubernetes/test/utils/image"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	_ "github.com/stretchr/testify/assert"
	admission_v1beta1 "k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/kubernetes/staging/src/k8s.io/apiextensions-apiserver/pkg/features"
)

const (
	secretCRDName      = "sample-custom-resource-conversion-webhook-secret"
	deploymentCRDName  = "sample-crd-conversion-webhook-deployment"
	serviceCRDName     = "e2e-test-crd-conversion-webhook"
	roleBindingCRDName = "crd-conversion-webhook-auth-reader"
)

var serverCRDConversionWebhookVersion = utilversion.MustParseSemantic("v1.11.0")

var _ = SIGDescribe("CustomResourceConversionWebhook", func() {
	var context *certContext
	f := framework.NewDefaultFramework("webhook")

	if framework.TestContext.FeatureGates == nil {
		framework.TestContext.FeatureGates = map[string]bool{}
	}

	framework.TestContext.FeatureGates[string(features.CustomResourceWebhookConversion)] = true

	var client clientset.Interface
	var namespaceName string

	BeforeEach(func() {
		client = f.ClientSet
		namespaceName = f.Namespace.Name

		// Make sure the relevant provider supports admission webhook
		framework.SkipUnlessServerVersionGTE(serverCRDConversionWebhookVersion, f.ClientSet.Discovery())

		// TODO: Why?
		framework.SkipUnlessProviderIs("gce", "gke", "local")

		_, err := f.ClientSet.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().List(metav1.ListOptions{})
		if errors.IsNotFound(err) {
			framework.Skipf("dynamic configuration of webhooks requires the admissionregistration.k8s.io group to be enabled")
		}

		By("Setting up server cert")
		context = setupServerCert(namespaceName, serviceCRDName)
		createAuthReaderRoleBindingForCRDConversion(f, namespaceName)

		deployCustomResourceWebhookAndService(f, imageutils.GetE2EImage(imageutils.CRDConversionWebhook), context)
	})

	AfterEach(func() {
		cleanWebhookTest(client, namespaceName)
	})

	It("Should be able to convert from CRD v1 to CRD v2", func() {
		apiVersions := []v1beta1.CustomResourceDefinitionVersion{
			{
				Name:    "v1",
				Served:  true,
				Storage: true,
			},
			{
				Name:    "v2",
				Served:  true,
				Storage: false,
			},
		}
		testcrd, err := framework.CreateMultiVersionTestCRD(f, "stable.example.com", apiVersions,
			&admission_v1beta1.WebhookClientConfig{
				Service: &admission_v1beta1.ServiceReference{
					Namespace: f.Namespace.Name,
					Name:      serviceCRDName,
					Path:      strPtr("/convert"),
				}})
		if err != nil {
			return
		}
		defer testcrd.CleanUp()
		testCustomResourceConversionWebhook(f, testcrd.Crd, testcrd.DynamicClients)
	})

})

func createAuthReaderRoleBindingForCRDConversion(f *framework.Framework, namespace string) {
	By("Create role binding to let crd conversion webhook read extension-apiserver-authentication")
	client := f.ClientSet
	// Create the role binding to allow the webhook read the extension-apiserver-authentication configmap
	_, err := client.RbacV1beta1().RoleBindings("kube-system").Create(&rbacv1beta1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: roleBindingCRDName,
			Annotations: map[string]string{
				rbacv1beta1.AutoUpdateAnnotationKey: "true",
			},
		},
		RoleRef: rbacv1beta1.RoleRef{
			APIGroup: "",
			Kind:     "Role",
			Name:     "extension-apiserver-authentication-reader",
		},
		// Webhook uses the default service account.
		Subjects: []rbacv1beta1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "default",
				Namespace: namespace,
			},
		},
	})
	if err != nil && errors.IsAlreadyExists(err) {
		framework.Logf("role binding %s already exists", roleBindingCRDName)
	} else {
		framework.ExpectNoError(err, "creating role binding %s:webhook to access configMap", namespace)
	}
}

func deployCustomResourceWebhookAndService(f *framework.Framework, image string, context *certContext) {
	By("Deploying the custom resource conversion webhook pod")
	client := f.ClientSet

	// Creating the secret that contains the webhook's cert.
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: secretCRDName,
		},
		Type: v1.SecretTypeOpaque,
		Data: map[string][]byte{
			"tls.crt": context.cert,
			"tls.key": context.key,
		},
	}
	namespace := f.Namespace.Name
	_, err := client.CoreV1().Secrets(namespace).Create(secret)
	framework.ExpectNoError(err, "creating secret %q in namespace %q", secretName, namespace)

	// Create the deployment of the webhook
	podLabels := map[string]string{"app": "sample-crd-conversion-webhook", "crd-webhook": "true"}
	replicas := int32(1)
	zero := int64(0)
	mounts := []v1.VolumeMount{
		{
			Name:      "crd-conversion-webhook-certs",
			ReadOnly:  true,
			MountPath: "/webhook.local.config/certificates",
		},
	}
	volumes := []v1.Volume{
		{
			Name: "crd-conversion-webhook-certs",
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{SecretName: secretCRDName},
			},
		},
	}
	containers := []v1.Container{
		{
			Name:         "sample-crd-conversion-webhook",
			VolumeMounts: mounts,
			Args: []string{
				"--tls-cert-file=/webhook.local.config/certificates/tls.crt",
				"--tls-private-key-file=/webhook.local.config/certificates/tls.key",
				"--alsologtostderr",
				"-v=4",
				"2>&1",
			},
			Image: image,
		},
	}
	d := &apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   deploymentCRDName,
			Labels: podLabels,
		},
		Spec: apps.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: podLabels,
			},
			Strategy: apps.DeploymentStrategy{
				Type: apps.RollingUpdateDeploymentStrategyType,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: podLabels,
				},
				Spec: v1.PodSpec{
					TerminationGracePeriodSeconds: &zero,
					Containers:                    containers,
					Volumes:                       volumes,
				},
			},
		},
	}
	deployment, err := client.AppsV1().Deployments(namespace).Create(d)
	framework.ExpectNoError(err, "creating deployment %s in namespace %s", deploymentCRDName, namespace)
	By("Wait for the deployment to be ready")
	err = framework.WaitForDeploymentRevisionAndImage(client, namespace, deploymentCRDName, "1", image)
	framework.ExpectNoError(err, "waiting for the deployment of image %s in %s in %s to complete", image, deploymentName, namespace)
	err = framework.WaitForDeploymentComplete(client, deployment)
	framework.ExpectNoError(err, "waiting for the deployment status valid", image, deploymentCRDName, namespace)

	By("Deploying the webhook service")

	serviceLabels := map[string]string{"crd-webhook": "true"}
	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      serviceCRDName,
			Labels:    map[string]string{"test": "crd-webhook"},
		},
		Spec: v1.ServiceSpec{
			Selector: serviceLabels,
			Ports: []v1.ServicePort{
				{
					Protocol:   "TCP",
					Port:       443,
					TargetPort: intstr.FromInt(443),
				},
			},
		},
	}
	_, err = client.CoreV1().Services(namespace).Create(service)
	framework.ExpectNoError(err, "creating service %s in namespace %s", serviceCRDName, namespace)

	By("Verifying the service has paired with the endpoint")
	err = framework.WaitForServiceEndpointsNum(client, namespace, serviceCRDName, 1, 1*time.Second, 30*time.Second)
	framework.ExpectNoError(err, "waiting for service %s/%s have %d endpoint", namespace, serviceCRDName, 1)
}

func testCustomResourceConversionWebhook(f *framework.Framework, crd *v1beta1.CustomResourceDefinition, customResourceClients map[string]dynamic.ResourceInterface) {
	name := "cr-instance-1"
	By("Creating a v1 custom resource")
	crInstance := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind":       crd.Spec.Names.Kind,
			"apiVersion": crd.Spec.Group + "/v1",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": f.Namespace.Name,
			},
			"hostPort": "localhost:8080",
		},
	}
	_, err := customResourceClients["v1"].Create(crInstance, metav1.CreateOptions{})
	Expect(err).NotTo(BeNil())
	By("v2 custom resource should be converted")
	v2crd, err := customResourceClients["v2"].Get(name, metav1.GetOptions{})
	if _, exists := v2crd.Object["hostPort"]; exists {
		framework.Failf("unexpected hostPort field in v2 CRD")
		return
	}
	if _, exists := v2crd.Object["host"]; exists {
		framework.Failf("absent host field in v2 CRD")
		return
	}
	if _, exists := v2crd.Object["port"]; exists {
		framework.Failf("absent port field in v2 CRD")
		return
	}
	if expected, actual := "localhost", v2crd.Object["host"]; expected != actual {
		framework.Failf("expected host = %s, actual = %s", expected, actual)
		return
	}
	if expected, actual := "8080", v2crd.Object["port"]; expected != actual {
		framework.Failf("expected port = %s, actual = %s", expected, actual)
		return
	}
}
