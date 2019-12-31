/*

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

package controllers

import (
	"context"
	"encoding/json"
	"math"
	"reflect"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	rolloutv1alpha1 "github.com/zeitgeistlabs/kubernetes-rollout/api/v1alpha1"
)

const (
	jobOwnerKey = ".metadata.controller"

	// Used to annotate ReplicaSets with the pod template.
	// This is a workaround to avoid mimicking PodTemplateSpec defaulting logic.
	specAnnotationKey = "pinneddeployment.rollout.zeitgeistlabs.io/template"
)

var (
	apiGroupVersionString = rolloutv1alpha1.GroupVersion.String()
)

// PinnedDeploymentReconciler reconciles a PinnedDeployment object
type PinnedDeploymentReconciler struct {
	client.Client
	Log    logr.Logger // TODO need to fake the log in tests.
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=rollout.zeitgeistlabs.io,resources=pinneddeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rollout.zeitgeistlabs.io,resources=pinneddeployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=replicasets,verbs=get;list;watch;create;update;patch;delete

func (r *PinnedDeploymentReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("pinneddeployment", req.NamespacedName)

	// Get object with data.
	var pinnedDeployment rolloutv1alpha1.PinnedDeployment
	if err := r.Get(ctx, req.NamespacedName, &pinnedDeployment); err != nil {
		log.Error(err, "unable to fetch PinnedDeployment")
		return ctrl.Result{}, client.IgnoreNotFound(err) // PinnedDeployment was deleted, do nothing.
		// If action (aside from child cleanup) is required, add finalizers.
	}

	var replicaSets appsv1.ReplicaSetList
	match := client.MatchingLabels{}
	for k, v := range pinnedDeployment.Labels {
		match[k] = v
	}

	// Fetch the ReplicaSets corresponding to the PinnedDeployment.
	if err := r.List(ctx, &replicaSets, client.InNamespace(req.Namespace), match); err != nil {
		log.Error(err, "unable to list matching ReplicaSets")
		return ctrl.Result{Requeue: true}, err
	}

	previousReplicaSet, nextReplicaSet, stray := r.sortReplicaSets(pinnedDeployment.Spec.Templates, replicaSets.Items)
	/*var previousName, nextName string
	if previousReplicaSet != nil {
		previousName = previousReplicaSet.Name
	}
	if nextReplicaSet != nil {
		nextName = nextReplicaSet.Name
	}
	r.Log.Info("found previous and next replicasets",
		"previous", previousName,
		"next", nextName)
	*/

	for _, strayRs := range stray {
		log.Info("Deleting stray ReplicaSet",
			"replicaset", strayRs.Name,
		)
		err := r.Delete(ctx, &strayRs)
		if err != nil {
			return ctrl.Result{Requeue: true}, err
		}
	}

	// Calculate previous + next target replicas.
	previousReplicas, nextReplicas := r.calculateReplicas(pinnedDeployment.Spec.Replicas, pinnedDeployment.Spec.ReplicasPercentNext, pinnedDeployment.Spec.ReplicasRoundingStrategy)

	// TODO make order dependent on ReplicasPercentNext change (IE, grow before shrink)
	err := r.makeReplicaSetConform(ctx, pinnedDeployment, previousReplicaSet, pinnedDeployment.Spec.Templates.Previous, previousReplicas)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}
	err = r.makeReplicaSetConform(ctx, pinnedDeployment, nextReplicaSet, pinnedDeployment.Spec.Templates.Next, nextReplicas)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	needsUpdate, err := r.updateStatus(&pinnedDeployment, previousReplicaSet, nextReplicaSet, previousReplicas, nextReplicas)
	if err != nil {
		return ctrl.Result{}, err
	}

	if needsUpdate {
		log.Info("Updating status.")

		// This operation must be last & conditional, as changing the status triggers a reconcile.
		if err := r.Client.Status().Update(ctx, &pinnedDeployment); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// Update status fields of the PinnedDeployment.
func (r *PinnedDeploymentReconciler) updateStatus(
	pinnedDeployment *rolloutv1alpha1.PinnedDeployment,
	previousReplicaSet *appsv1.ReplicaSet,
	nextReplicaSet *appsv1.ReplicaSet,
	previousReplicaTarget int32,
	nextReplicasTarget int32,
) (bool, error) {

	var originalStatus = *pinnedDeployment.Status.DeepCopy() // Save the original status.

	// Write replica targets.
	pinnedDeployment.Status.Versions.Previous.DesiredReplicas = previousReplicaTarget
	pinnedDeployment.Status.Versions.Next.DesiredReplicas = nextReplicasTarget

	currentReplicas := map[string]int32{}
	readyReplicas := map[string]int32{}
	for key, rs := range map[string]*appsv1.ReplicaSet{"previous": previousReplicaSet, "next": nextReplicaSet} {
		if rs != nil {
			currentReplicas[key] = rs.Status.Replicas
			readyReplicas[key] = rs.Status.ReadyReplicas
		}
	}

	// Set versioned statuses.
	for key, versionStatus := range map[string]*rolloutv1alpha1.PinnedDeploymentVersionedStatus{
		"previous": &pinnedDeployment.Status.Versions.Previous,
		"next":     &pinnedDeployment.Status.Versions.Next,
	} {
		// Deliberately use the 0 from a not-found.
		versionStatus.CurrentReplicas, _ = currentReplicas[key]
		versionStatus.ReadyReplicas, _ = readyReplicas[key]
	}

	// Set overall replica statuses.
	pinnedDeployment.Status.CurrentReplicas = totalValueForKeys(currentReplicas)
	pinnedDeployment.Status.ReadyReplicas = totalValueForKeys(readyReplicas)

	// Check if Status.Selector needs an update.
	selector, err := metav1.LabelSelectorAsSelector(pinnedDeployment.Spec.Selector)
	if err != nil {
		return false, err
	}
	pinnedDeployment.Status.Selector = selector.String()

	return !reflect.DeepEqual(originalStatus, pinnedDeployment.Status), nil
}

// Ensure that a ReplicaSet exists, and is in the desired state.
func (r *PinnedDeploymentReconciler) makeReplicaSetConform(
	ctx context.Context,
	pinnedDeployment rolloutv1alpha1.PinnedDeployment,
	existingReplicaSet *appsv1.ReplicaSet,
	podTemplateSpec rolloutv1alpha1.FakePodTemplateSpec,
	replicas int32,
) error {

	// Create missing ReplicaSet.
	if existingReplicaSet == nil {
		nextReplicaSet, err := r.buildReplicaSet(
			pinnedDeployment.Namespace,
			pinnedDeployment.Name,
			*pinnedDeployment.Spec.Selector,
			podTemplateSpec,
			replicas,
		)
		if err != nil {
			return err
		}

		r.Log.Info("Creating new ReplicaSet",
			"replicaset", nextReplicaSet.Name,
		)

		if err := controllerutil.SetControllerReference(&pinnedDeployment, nextReplicaSet, r.Scheme); err != nil {
			return errors.Wrap(err, "failed to set controller reference")
		}

		err = r.Client.Create(ctx, nextReplicaSet)
		if err != nil {
			r.Log.Info("Error creating new ReplicaSet",
				"replicaset", nextReplicaSet.Name,
				"error", err,
			)
			return err
		}

		r.Log.Info("Created new ReplicaSet",
			"replicaset", nextReplicaSet.Name,
		)

	} else { // Ensure that the ReplicaSet has the desired state.
		desiredReplicaSetSpec := r.buildReplicaSetSpec(
			*pinnedDeployment.Spec.Selector,
			podTemplateSpec,
			replicas,
		)

		if !r.selectFieldsReplicaSetSpecMatch(existingReplicaSet.Spec, desiredReplicaSetSpec) {
			r.Log.Info("Updating ReplicaSet spec",
				"replicaset", existingReplicaSet.Name,
				"replicas", replicas,
				"oldReplicas", existingReplicaSet.Spec.Replicas,
			)

			existingReplicaSet.Spec = desiredReplicaSetSpec

			err := r.Client.Update(ctx, existingReplicaSet)
			if err != nil {
				r.Log.Info("Error updating ReplicaSet",
					"replicaset", existingReplicaSet.Name,
					"error", err,
				)
				return err
			}

			r.Log.Info("Updated ReplicaSet",
				"replicaset", existingReplicaSet.Name,
			)
		}
	}

	return nil
}

func (r *PinnedDeploymentReconciler) buildReplicaSet(
	pinnedDeploymentNamespace string,
	pinnedDeploymentName string,
	pinnedDeploymentSelector metav1.LabelSelector,
	podTemplateSpec rolloutv1alpha1.FakePodTemplateSpec,
	replicas int32,
) (*appsv1.ReplicaSet, error) {

	jsonSpec, err := r.specToJson(podTemplateSpec)
	if err != nil {
		return nil, err
	}

	// TODO is this right?
	labels := map[string]string{}
	for k, v := range pinnedDeploymentSelector.MatchLabels {
		labels[k] = v
	}

	replicaSet := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: pinnedDeploymentNamespace,
			Name:      pinnedDeploymentName + "-" + randomString(3),
			Annotations: map[string]string{
				specAnnotationKey: jsonSpec,
			},
			Labels: labels,
		},
		Spec: r.buildReplicaSetSpec(
			pinnedDeploymentSelector,
			podTemplateSpec,
			replicas,
		),
	}

	return replicaSet, nil
}

func (r *PinnedDeploymentReconciler) buildReplicaSetSpec(selector metav1.LabelSelector, podTemplateSpec rolloutv1alpha1.FakePodTemplateSpec, replicas int32) appsv1.ReplicaSetSpec {
	return appsv1.ReplicaSetSpec{
		Replicas: &replicas,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: podTemplateSpec.Metadata.Labels,
			},
			Spec: podTemplateSpec.Spec,
		},
		Selector:        &selector,
		MinReadySeconds: 0,
	}
}

// Checks that certain fields of a pair of ReplicaSets match.
// In particular, this ignores the PodTemplateSpec, as the specs are mutated by Kubernetes admission controllers.
func (r *PinnedDeploymentReconciler) selectFieldsReplicaSetSpecMatch(a, b appsv1.ReplicaSetSpec) bool {
	return a.Replicas != nil && b.Replicas != nil && *a.Replicas == *b.Replicas &&
		a.MinReadySeconds == b.MinReadySeconds &&
		reflect.DeepEqual(*a.Selector, *b.Selector)
}

// Hack to get around PodSpec defaulting logic.
// The controller stores an annotation containing the spec, which we compare to here.
// The annotation, unlike the actual ReplicaSet spec, will match the spec that we generated.
func (r *PinnedDeploymentReconciler) replicaSetAnnotationMatchesSpec(original rolloutv1alpha1.FakePodTemplateSpec, observed appsv1.ReplicaSet) bool {
	annotation, found := observed.Annotations[specAnnotationKey]
	if !found {
		return false
	}

	rehydratedSpec, err := r.jsonToSpec(annotation)
	if err != nil {
		return false
	}

	// Use object comparison, as maps don't serialize deterministically.
	// TODO there's probably a better way to do this.
	return fakePodTemplateSpecEqual(original, rehydratedSpec)
}

func (r *PinnedDeploymentReconciler) jsonToSpec(str string) (rolloutv1alpha1.FakePodTemplateSpec, error) {
	var spec rolloutv1alpha1.FakePodTemplateSpec
	err := json.Unmarshal([]byte(str), &spec)
	return spec, err
}

func (r *PinnedDeploymentReconciler) specToJson(spec rolloutv1alpha1.FakePodTemplateSpec) (string, error) {
	b, err := json.Marshal(spec)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func (r *PinnedDeploymentReconciler) calculateReplicas(replicas int32, percentNew int32, replicasRounding rolloutv1alpha1.ReplicasRoundingStrategyType) (int32, int32) {
	target := float64(replicas) * (float64(percentNew) / 100)

	if replicasRounding == rolloutv1alpha1.DownReplicasRoundingStrategyType {

		nextReplicas := int32(math.Floor(target))
		return int32(replicas - nextReplicas), int32(nextReplicas)

	} else if replicasRounding == rolloutv1alpha1.UpReplicasRoundingStrategyType {

		nextReplicas := int32(math.Ceil(target))
		return int32(replicas - nextReplicas), int32(nextReplicas)

	} else { // Nearest / fallback

		nextReplicas := int32(math.Round(target))
		return int32(replicas - nextReplicas), int32(nextReplicas)

	}
}

func fakePodTemplateSpecEqual(copy1, copy2 rolloutv1alpha1.FakePodTemplateSpec) bool {
	return apiequality.Semantic.DeepEqual(copy1, copy2)
}

// Sort all matching ReplicaSets into the previous, next, and others.
func (r *PinnedDeploymentReconciler) sortReplicaSets(templates rolloutv1alpha1.PreviousNextPodTemplateSpecPair, replicaSets []appsv1.ReplicaSet) (*appsv1.ReplicaSet, *appsv1.ReplicaSet, []appsv1.ReplicaSet) {
	var previousReplicaSet *appsv1.ReplicaSet = nil
	var nextReplicaSet *appsv1.ReplicaSet = nil
	otherRs := make([]appsv1.ReplicaSet, 0)

	// Use PodTemplateSpec to identify which ReplicaSet is which.
	for i := range replicaSets {
		if r.replicaSetAnnotationMatchesSpec(templates.Previous, replicaSets[i]) {
			previousReplicaSet = &replicaSets[i]
		} else if r.replicaSetAnnotationMatchesSpec(templates.Next, replicaSets[i]) {
			nextReplicaSet = &replicaSets[i]
		} else {
			otherRs = append(otherRs, replicaSets[i])
		}
	}

	if previousReplicaSet != nil && nextReplicaSet != nil && previousReplicaSet.Name == nextReplicaSet.Name {
		r.Log.Info("Matched the same replicaset for previous and next specs",
			"replicaset", previousReplicaSet.Name)
	}

	return previousReplicaSet, nextReplicaSet, otherRs
}

func (r *PinnedDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(&appsv1.ReplicaSet{}, jobOwnerKey, func(rawObj runtime.Object) []string {
		rs := rawObj.(*appsv1.ReplicaSet)
		owner := metav1.GetControllerOf(rs)
		if owner == nil {
			return nil
		}
		if owner.APIVersion != apiGroupVersionString || owner.Kind != "PinnedDeployment" {
			return nil
		}

		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&rolloutv1alpha1.PinnedDeployment{}).
		Owns(&appsv1.ReplicaSet{}).
		Complete(r)
}
