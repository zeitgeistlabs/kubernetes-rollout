package controllers

import (
	"reflect"
	"testing"

	// Test utils.
	/*
		. "github.com/onsi/ginkgo"
		. "github.com/onsi/gomega"
	*/

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/zeitgeistlabs/kubernetes-rollout/api/v1alpha1"
	rolloutv1alpha1 "github.com/zeitgeistlabs/kubernetes-rollout/api/v1alpha1"
)

func TestPinnedDeploymentReconciler_updateStatus(t *testing.T) {
	pdNoStatus := rolloutv1alpha1.PinnedDeployment{
		Spec: rolloutv1alpha1.PinnedDeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"key": "value",
				},
			},
		},
	}

	cases := []struct {
		pd                rolloutv1alpha1.PinnedDeployment
		previousRs        *v1.ReplicaSet
		nextRs            *v1.ReplicaSet
		previousReplicas  int32
		nextReplicas      int32
		expectStatus      rolloutv1alpha1.PinnedDeploymentStatus
		expectNeedsUpdate bool
	}{
		{
			pd:                pdNoStatus,
			previousRs:        nil,
			nextRs:            nil,
			previousReplicas:  4,
			nextReplicas:      1,
			expectNeedsUpdate: true,
			expectStatus: rolloutv1alpha1.PinnedDeploymentStatus{
				Selector:        "key=value",
				CurrentReplicas: 0,
				ReadyReplicas:   0,
				Versions: rolloutv1alpha1.PinnedDeploymentVersionedStatuses{
					Previous: rolloutv1alpha1.PinnedDeploymentVersionedStatus{
						DesiredReplicas: 4,
						CurrentReplicas: 0,
						ReadyReplicas:   0,
					},
					Next: rolloutv1alpha1.PinnedDeploymentVersionedStatus{
						DesiredReplicas: 1,
						CurrentReplicas: 0,
						ReadyReplicas:   0,
					},
				},
			},
		},
		{
			pd: pdNoStatus,
			previousRs: &v1.ReplicaSet{
				Status: v1.ReplicaSetStatus{
					Replicas:      9,
					ReadyReplicas: 8,
				},
			},
			nextRs:            nil,
			previousReplicas:  10,
			nextReplicas:      10,
			expectNeedsUpdate: true,
			expectStatus: rolloutv1alpha1.PinnedDeploymentStatus{
				Selector:        "key=value",
				CurrentReplicas: 9,
				ReadyReplicas:   8,
				Versions: rolloutv1alpha1.PinnedDeploymentVersionedStatuses{
					Previous: rolloutv1alpha1.PinnedDeploymentVersionedStatus{
						DesiredReplicas: 10,
						CurrentReplicas: 9,
						ReadyReplicas:   8,
					},
					Next: rolloutv1alpha1.PinnedDeploymentVersionedStatus{
						DesiredReplicas: 10,
						CurrentReplicas: 0,
						ReadyReplicas:   0,
					},
				},
			},
		},
		{
			pd:         pdNoStatus,
			previousRs: nil,
			nextRs: &v1.ReplicaSet{
				Status: v1.ReplicaSetStatus{
					Replicas:      9,
					ReadyReplicas: 8,
				},
			},
			previousReplicas:  10,
			nextReplicas:      10,
			expectNeedsUpdate: true,
			expectStatus: rolloutv1alpha1.PinnedDeploymentStatus{
				Selector:        "key=value",
				CurrentReplicas: 9,
				ReadyReplicas:   8,
				Versions: rolloutv1alpha1.PinnedDeploymentVersionedStatuses{
					Previous: rolloutv1alpha1.PinnedDeploymentVersionedStatus{
						DesiredReplicas: 10,
						CurrentReplicas: 0,
						ReadyReplicas:   0,
					},
					Next: rolloutv1alpha1.PinnedDeploymentVersionedStatus{
						DesiredReplicas: 10,
						CurrentReplicas: 9,
						ReadyReplicas:   8,
					},
				},
			},
		},
		{
			pd: pdNoStatus,
			previousRs: &v1.ReplicaSet{
				Status: v1.ReplicaSetStatus{
					Replicas:      5,
					ReadyReplicas: 5,
				},
			},
			nextRs: &v1.ReplicaSet{
				Status: v1.ReplicaSetStatus{
					Replicas:      15,
					ReadyReplicas: 12,
				},
			},
			previousReplicas:  5,
			nextReplicas:      15,
			expectNeedsUpdate: true,
			expectStatus: rolloutv1alpha1.PinnedDeploymentStatus{
				Selector:        "key=value",
				CurrentReplicas: 20,
				ReadyReplicas:   17,
				Versions: rolloutv1alpha1.PinnedDeploymentVersionedStatuses{
					Previous: rolloutv1alpha1.PinnedDeploymentVersionedStatus{
						DesiredReplicas: 5,
						CurrentReplicas: 5,
						ReadyReplicas:   5,
					},
					Next: rolloutv1alpha1.PinnedDeploymentVersionedStatus{
						DesiredReplicas: 15,
						CurrentReplicas: 15,
						ReadyReplicas:   12,
					},
				},
			},
		},
		{
			pd: rolloutv1alpha1.PinnedDeployment{
				Spec: rolloutv1alpha1.PinnedDeploymentSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"key": "value",
						},
					},
				},
				Status: rolloutv1alpha1.PinnedDeploymentStatus{
					Selector:        "key=value",
					CurrentReplicas: 0,
					ReadyReplicas:   0,
					Versions: rolloutv1alpha1.PinnedDeploymentVersionedStatuses{
						Previous: rolloutv1alpha1.PinnedDeploymentVersionedStatus{
							DesiredReplicas: 9,
							CurrentReplicas: 0,
							ReadyReplicas:   0,
						},
						Next: rolloutv1alpha1.PinnedDeploymentVersionedStatus{
							DesiredReplicas: 1,
							CurrentReplicas: 0,
							ReadyReplicas:   0,
						},
					},
				},
			},
			previousRs:        nil,
			nextRs:            nil,
			previousReplicas:  9,
			nextReplicas:      1,
			expectNeedsUpdate: false,
			expectStatus: rolloutv1alpha1.PinnedDeploymentStatus{
				Selector:        "key=value",
				CurrentReplicas: 0,
				ReadyReplicas:   0,
				Versions: rolloutv1alpha1.PinnedDeploymentVersionedStatuses{
					Previous: rolloutv1alpha1.PinnedDeploymentVersionedStatus{
						DesiredReplicas: 9,
						CurrentReplicas: 0,
						ReadyReplicas:   0,
					},
					Next: rolloutv1alpha1.PinnedDeploymentVersionedStatus{
						DesiredReplicas: 1,
						CurrentReplicas: 0,
						ReadyReplicas:   0,
					},
				},
			},
		},
	}

	for _, c := range cases {
		r := PinnedDeploymentReconciler{}

		needsUpdate, err := r.updateStatus(&c.pd, c.previousRs, c.nextRs, c.previousReplicas, c.nextReplicas)
		if err != nil {
			t.Errorf("expected no error: %v", err)
		}

		if needsUpdate != c.expectNeedsUpdate {
			t.Errorf("expected 'needs update' to be %v, got %v", c.expectNeedsUpdate, needsUpdate)
		}

		if !reflect.DeepEqual(c.pd.Status, c.expectStatus) {
			t.Errorf("Statuses didn't match. GOT: %v\nEXPECTED: %v", c.pd.Status, c.expectStatus)
		}
	}

}

func TestPinnedDeploymentReconciler_setControllerReference(t *testing.T) {
	pd := v1alpha1.PinnedDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: v1alpha1.PinnedDeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
			Replicas:            2,
			ReplicasPercentNext: 50,
			Templates: v1alpha1.PreviousNextPodTemplateSpecPair{
				Previous: v1alpha1.FakePodTemplateSpec{
					Metadata: v1alpha1.BasicMetadata{
						Labels: map[string]string{
							"app": "test",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "example",
								Image: "example:1",
							},
						},
					},
				},
				Next: v1alpha1.FakePodTemplateSpec{
					Metadata: v1alpha1.BasicMetadata{
						Labels: map[string]string{
							"app": "test",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "example",
								Image: "example:2",
							},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = rolloutv1alpha1.AddToScheme(scheme)

	r := PinnedDeploymentReconciler{
		Scheme: scheme,
	}
	rs, _ := r.buildReplicaSet(pd.Namespace, pd.Name,
		*pd.Spec.Selector, pd.Spec.Templates.Previous, 1)

	if err := controllerutil.SetControllerReference(&pd, rs, r.Scheme); err != nil {
		t.Errorf("failed to set controller reference: %v", err)
	}
}

func TestPinnedDeploymentReconciler_sortReplicaSets(t *testing.T) {
	r := PinnedDeploymentReconciler{}

	cases := []struct {
		description        string
		templates          v1alpha1.PreviousNextPodTemplateSpecPair
		replicaSets        []v1.ReplicaSet
		expectNextName     string
		expectPreviousName string
		expectOtherNames   []string
	}{
		{
			description: "no ReplicaSets",
			templates: v1alpha1.PreviousNextPodTemplateSpecPair{
				Previous: v1alpha1.FakePodTemplateSpec{
					Metadata: v1alpha1.BasicMetadata{
						Labels: map[string]string{
							"app": "demo",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "example",
								Image: "example:1",
							},
						},
					},
				},
				Next: v1alpha1.FakePodTemplateSpec{
					Metadata: v1alpha1.BasicMetadata{
						Labels: map[string]string{
							"app": "demo",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "example",
								Image: "example:2",
							},
						},
					},
				},
			},
			replicaSets:        []v1.ReplicaSet{},
			expectPreviousName: "",
			expectNextName:     "",
			expectOtherNames:   []string{},
		},
		{
			description: "only old",
			templates: v1alpha1.PreviousNextPodTemplateSpecPair{
				Previous: v1alpha1.FakePodTemplateSpec{
					Metadata: v1alpha1.BasicMetadata{
						Labels: map[string]string{
							"app": "demo",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "example",
								Image: "example:1",
							},
						},
					},
				},
				Next: v1alpha1.FakePodTemplateSpec{
					Metadata: v1alpha1.BasicMetadata{
						Labels: map[string]string{
							"app": "demo",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "example",
								Image: "example:2",
							},
						},
					},
				},
			},
			replicaSets: []v1.ReplicaSet{
				replicaSetOnly(r.buildReplicaSet(
					"old",
					"demo",
					metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "demo",
						},
					},
					v1alpha1.FakePodTemplateSpec{
						Metadata: v1alpha1.BasicMetadata{
							Labels: map[string]string{
								"app": "demo",
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "example",
									Image: "example:1",
								},
							},
						},
					},
					0,
				)),
			},
			expectPreviousName: "old",
			expectNextName:     "",
			expectOtherNames:   []string{},
		},
		{
			description: "only old and new",
			templates: v1alpha1.PreviousNextPodTemplateSpecPair{
				Previous: v1alpha1.FakePodTemplateSpec{
					Metadata: v1alpha1.BasicMetadata{
						Labels: map[string]string{
							"app": "demo",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "example",
								Image: "example:1",
							},
						},
					},
				},
				Next: v1alpha1.FakePodTemplateSpec{
					Metadata: v1alpha1.BasicMetadata{
						Labels: map[string]string{
							"app": "demo",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "example",
								Image: "example:2",
							},
						},
					},
				},
			},
			replicaSets: []v1.ReplicaSet{
				replicaSetOnly(r.buildReplicaSet(
					"newrs",
					"demo",
					metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "demo",
						},
					},
					v1alpha1.FakePodTemplateSpec{
						Metadata: v1alpha1.BasicMetadata{
							Labels: map[string]string{
								"app": "demo",
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "example",
									Image: "example:2",
								},
							},
						},
					},
					0,
				)),
				replicaSetOnly(r.buildReplicaSet(
					"oldrs",
					"demo",
					metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "demo",
						},
					},
					v1alpha1.FakePodTemplateSpec{
						Metadata: v1alpha1.BasicMetadata{
							Labels: map[string]string{
								"app": "demo",
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "example",
									Image: "example:1",
								},
							},
						},
					},
					0,
				)),
			},
			expectPreviousName: "oldrs",
			expectNextName:     "newrs",
			expectOtherNames:   []string{},
		},
		{
			description: "old and others",
			templates: v1alpha1.PreviousNextPodTemplateSpecPair{
				Previous: v1alpha1.FakePodTemplateSpec{
					Metadata: v1alpha1.BasicMetadata{
						Labels: map[string]string{
							"app": "demo",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "example",
								Image: "example:1",
							},
						},
					},
				},
				Next: v1alpha1.FakePodTemplateSpec{
					Metadata: v1alpha1.BasicMetadata{
						Labels: map[string]string{
							"app": "demo",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "example",
								Image: "example:2",
							},
						},
					},
				},
			},
			replicaSets: []v1.ReplicaSet{
				replicaSetOnly(r.buildReplicaSet(
					"other1",
					"demo",
					metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "demo",
						},
					},
					v1alpha1.FakePodTemplateSpec{
						Metadata: v1alpha1.BasicMetadata{
							Labels: map[string]string{
								"app": "demo",
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "not-example",
									Image: "example:2",
								},
							},
						},
					},
					0,
				)),
				replicaSetOnly(r.buildReplicaSet(
					"oldrs",
					"demo",
					metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "demo",
						},
					},
					v1alpha1.FakePodTemplateSpec{
						Metadata: v1alpha1.BasicMetadata{
							Labels: map[string]string{
								"app": "demo",
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "example",
									Image: "example:1",
								},
							},
						},
					},
					0,
				)),
				replicaSetOnly(r.buildReplicaSet(
					"other2",
					"demo",
					metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "demo",
						},
					},
					v1alpha1.FakePodTemplateSpec{
						Metadata: v1alpha1.BasicMetadata{
							Labels: map[string]string{
								"app": "demo",
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "not-example",
									Image: "example:2",
								},
							},
						},
					},
					0,
				)),
			},
			expectPreviousName: "oldrs",
			expectNextName:     "",
			expectOtherNames:   []string{"other1", "other2"},
		},
	}

	for _, c := range cases {
		previousReplicaSet, nextReplicaSet, others := r.sortReplicaSets(c.templates, c.replicaSets)
		// Hack for making the test input simpler: use ReplicaSet.Namespace as a ID for each ReplicaSet (reminder: ReplicaSet.Name is partially auto-generated)
		previousName := ""
		if previousReplicaSet != nil {
			previousName = previousReplicaSet.Namespace
		}
		nextName := ""
		if nextReplicaSet != nil {
			nextName = nextReplicaSet.Namespace
		}

		if previousName != c.expectPreviousName {
			t.Errorf("case '%s' matched '%s' for previous, expected '%s'", c.description, previousName, c.expectPreviousName)
		}

		if nextName != c.expectNextName {
			t.Errorf("case '%s' matched '%s' for next, expected '%s'", c.description, nextName, c.expectNextName)
		}

		otherNames := make([]string, len(others))
		for i, otherReplicaSet := range others {
			otherNames[i] = otherReplicaSet.Namespace
		}

		if !reflect.DeepEqual(otherNames, c.expectOtherNames) {
			t.Errorf("case '%s' matched '%s' for others, expected '%s'", c.description, otherNames, c.expectOtherNames)
		}
	}
}

func TestPinnedDeploymentReconciler_replicaSetMatchesSpec(t *testing.T) {
	cases := []struct {
		spec          v1alpha1.FakePodTemplateSpec
		annotation    string
		expectSuccess bool
	}{
		{
			spec: v1alpha1.FakePodTemplateSpec{
				Metadata: v1alpha1.BasicMetadata{
					Labels: map[string]string{
						"role": "canary",
						"app":  "demo",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "example",
							Image: "example:1",
						},
						{
							Name:  "sidecar",
							Image: "sidecar:1",
						},
					},
				},
			},
			annotation:    `{"metadata":{"labels":{"app":"demo","role":"canary"}},"spec":{"containers":[{"name":"example","image":"example:1","resources":{}},{"name":"sidecar","image":"sidecar:1","resources":{}}]}}`,
			expectSuccess: true,
		},
		{
			spec: v1alpha1.FakePodTemplateSpec{
				Metadata: v1alpha1.BasicMetadata{
					Labels: map[string]string{
						"role": "canary",
						"app":  "demo",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "example",
							Image: "example:2",
						},
						{
							Name:  "sidecar",
							Image: "sidecar:1",
						},
					},
				},
			},
			annotation:    `{"metadata":{"labels":{"app":"demo","role":"canary"}},"spec":{"containers":[{"name":"example","image":"example:1","resources":{}},{"name":"sidecar","image":"sidecar:1","resources":{}}]}}`,
			expectSuccess: false,
		},
	}
	r := PinnedDeploymentReconciler{}

	for _, c := range cases {
		rs := v1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					specAnnotationKey: c.annotation,
				},
			},
		}

		result := r.replicaSetAnnotationMatchesSpec(c.spec, rs)
		if result != c.expectSuccess {
			t.Errorf("not equal")
		}
	}
}

func TestPinnedDeploymentReconciler_calculateReplicas(t *testing.T) {
	cases := []struct {
		replicas           int32
		percentNew         int32
		replicasRounding   v1alpha1.ReplicasRoundingStrategyType
		expectNextReplicas int32
	}{
		{
			replicasRounding:   v1alpha1.DownReplicasRoundingStrategyType,
			replicas:           5,
			percentNew:         15,
			expectNextReplicas: 0,
		},
		{
			replicasRounding:   v1alpha1.DownReplicasRoundingStrategyType,
			replicas:           5,
			percentNew:         80,
			expectNextReplicas: 4,
		},
		{
			replicasRounding:   v1alpha1.UpReplicasRoundingStrategyType,
			replicas:           5,
			percentNew:         15,
			expectNextReplicas: 1,
		},
		{
			replicasRounding:   v1alpha1.UpReplicasRoundingStrategyType,
			replicas:           5,
			percentNew:         80,
			expectNextReplicas: 4,
		},
		{
			replicasRounding:   v1alpha1.NearestReplicasRoundingStrategyType,
			replicas:           5,
			percentNew:         45,
			expectNextReplicas: 2,
		},
		{
			replicasRounding:   v1alpha1.NearestReplicasRoundingStrategyType,
			replicas:           5,
			percentNew:         50,
			expectNextReplicas: 3,
		},
		{
			replicasRounding:   v1alpha1.NearestReplicasRoundingStrategyType,
			replicas:           5,
			percentNew:         55,
			expectNextReplicas: 3,
		},
	}

	for _, c := range cases {
		r := PinnedDeploymentReconciler{}
		actualOld, actualNew := r.calculateReplicas(c.replicas, c.percentNew, c.replicasRounding)

		expectOld := int32(c.replicas) - c.expectNextReplicas
		if actualOld != expectOld {
			t.Errorf("%d replicas %d percentNew (%s): got %d old replicas, expected %d", c.replicas, c.percentNew, c.replicasRounding, actualOld, expectOld)
		}

		if actualNew != c.expectNextReplicas {
			t.Errorf("%d replicas %d percentNew (%s): got %d new replicas, expected %d", c.replicas, c.percentNew, c.replicasRounding, actualNew, c.expectNextReplicas)
		}
	}
}

/*
var _ = Describe("PinnedDeployments Controller", func() {

	const timeout = time.Second * 30
	const interval = time.Second * 1

	AfterEach(func() {
		// Add any teardown steps that needs to be executed after each test
	})

	Context("Test", func() {
		It("Should create new resources successfully", func() {
			pdName := "demo"

			spec := v1alpha1.PinnedDeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "nginx",
					},
				},
				Replicas:                 10,
				ReplicasPercentNext:       10,
				ReplicasRoundingStrategy: v1alpha1.NearestReplicasRoundingStrategyType,
				Templates: v1alpha1.PreviousNextPodTemplateSpecPair{
					Previous: v1alpha1.FakePodTemplateSpec{
						Metadata: v1alpha1.BasicMetadata{
							Labels: map[string]string{
								"app": "nginx",
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "example",
									Image: "example:1",
								},
							},
						},
					},
					Next: v1alpha1.FakePodTemplateSpec{
						Metadata: v1alpha1.BasicMetadata{
							Labels: map[string]string{
								"app": "nginx",
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "example",
									Image: "example:2",
								},
							},
						},
					},
				},
			}

			key := types.NamespacedName{
				Name:      pdName,
				Namespace: "default",
			}

			toCreate := &v1alpha1.PinnedDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: spec,
			}

			By("Creating the resource")
			Expect(k8sClient.Create(context.Background(), toCreate)).Should(Succeed())
			time.Sleep(time.Second * 5)

			fetched := &v1alpha1.PinnedDeployment{}
			Eventually(func() bool {
				k8sClient.Get(context.Background(), key, fetched)
				return true // Hmm?
			}, timeout, interval).Should(BeTrue())

			// TODO expand
		})
	})
})
*/
